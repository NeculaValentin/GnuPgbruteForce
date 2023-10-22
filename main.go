package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const maxPwdLength = 5 //the max password length (more than 5 will take too long)

// A PromptFunction is used as a callback by functions that may need to decrypt
// a private key, or prompt for a passphrase. If the decrypted private key or given passphrase isn't
// correct, the function will be called again, forever. That's why i have the flag called.
func decrypt(data []byte, pwd []byte) bool { //function to decrypt the data with the pwd
	buffer := bytes.NewBuffer(data) //create a buffer with the data

	called := false                                                      //variable to check if the prompt function was called
	prompt := func(keys []openpgp.Key, symmetric bool) ([]byte, error) { //function to prompt for the pwd
		if called {
			return nil, fmt.Errorf("one attmept only")
		}
		called = true
		return pwd, nil
	}

	_, err := openpgp.ReadMessage(buffer, nil, prompt, nil) //try to decrypt the data, this function
	isDecryptError := err != nil
	return !isDecryptError
}

func worker(data []byte, wg *sync.WaitGroup, pwdCh <-chan string, resCh chan<- string) {
	defer wg.Done() //when the function finishes, put the worker in done
	for pwd := range pwdCh {
		if decrypt(data, []byte(pwd)) {
			resCh <- pwd
			return
		}
	}
}

// this is a recursive function that generates all possible passwords
// it starts with a length of 1 and goes up to the max password length
// it generates all possible passwords with the current length
// then it calls itself with the next length
// it stops when it reaches the max password length
// it sends the passwords to the channel
// it closes the channel when it finishes
func pwdGenerator(ch chan<- string) {
	const chars = "abcdefghijklmnopqrstuvwxyz"   //the characters to use in the passwords (ascii lowercase letters, from the challenge)
	var generate func([]byte, int)               //the function to generate the passwords
	generate = func(prefix []byte, length int) { //the function to generate the passwords
		if length == 0 {
			pwd := string(prefix)
			ch <- pwd
			return
		}
		for _, c := range chars {
			newPrefix := append(append([]byte{}, prefix...), byte(c)) //append the character to the prefix
			generate(newPrefix, length-1)                             //call the function with the new prefix and the length - 1
		}
	}
	for length := 1; length <= maxPwdLength; length++ { //loop from 1 to the max password length
		generate([]byte{}, length) //call the function with an empty prefix and the length
	}
	close(ch) //close the channel
}

func main() {
	start := time.Now()

	debug.SetGCPercent(-1) // Disable Garbage Collector for performance improvements

	if len(os.Args) < 2 { //file path as argument
		log.Fatal("Provide the path of the file to decrypt")
	}

	path := os.Args[1]             //get the file path from the console
	data, err := os.ReadFile(path) //read the file
	if err != nil {                //check for errors
		log.Fatal("Error while parsing the file: ", err)
	}

	//get the number of cores
	//not using more because it uses IO, and having more workers than cores will not increase performance
	workers := runtime.NumCPU()

	//create a channel with a buffer of 1000 to store the passwords
	//this ensures that the workers will not wait for the password generator to generate a password
	//this also ensures that the password generator will not wait for the workers to finish
	//no need to make it bigger cause the decryption takes more time than the password generation
	pwdCh := make(chan string, 1000)

	//create a channel with a buffer of 1 where the workers will send the result
	resCh := make(chan string, 1)

	//create a wait group to wait for the workers to finish
	var wg sync.WaitGroup

	//start the workers and add them to the wait group
	//this ensures that the program will not exit before the workers finish
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(data, &wg, pwdCh, resCh)
	}

	//start the password generator
	//no need to add it to the wait group because if the password is found before the generator finishes, the program needs to exit
	go pwdGenerator(pwdCh)

	//select statement to start the goroutine to wait for the wait group to finish
	select {
	case pwd := <-resCh: //wait for the result
		fmt.Println("Password found:", pwd)
	case <-func() chan struct{} { //wait for the wait group to finish
		ch := make(chan struct{}) //create a channel to return
		go func() {               //start a goroutine to wait for the wait group to finish
			wg.Wait() //wait for the wait group to finish
			close(ch) //close the channel to return
		}() //start the goroutine
		return ch //return the channel
	}(): //if the wait group finishes without finding the password is found
		fmt.Println("No password found")
	}

	endTime := time.Since(start)
	fmt.Printf("Time to execute:  %s\n", endTime)
}