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

const maxPasswordLength = 5 //the max password length (more than 5 will take too long)

//this function decrypts the data with the password
//

func decrypt(data []byte, password []byte) bool { //function to decrypt the data with the password
	decoder := bytes.NewBuffer(data) //create a buffer with the data

	called := false                                                      //variable to check if the prompt function was called
	prompt := func(keys []openpgp.Key, symmetric bool) ([]byte, error) { //function to prompt for the password
		if called {
			return nil, fmt.Errorf("single attempt made")
		}
		called = true

		return password, nil
	}

	_, err := openpgp.ReadMessage(decoder, nil, prompt, nil) //try to decrypt the data, this function
	isDecryptError := err != nil
	return !isDecryptError
}

func worker(data []byte, ch <-chan string, wg *sync.WaitGroup, resultCh chan<- string) {
	defer wg.Done()
	for password := range ch {
		if decrypt(data, []byte(password)) {
			resultCh <- password
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
func generatePasswords(ch chan<- string) {
	const chars = "abcdefghijklmnopqrstuvwxyz"   //the characters to use in the passwords (ascii lowercase letters, from the challenge)
	var generate func([]byte, int)               //the function to generate the passwords
	generate = func(prefix []byte, length int) { //the function to generate the passwords
		if length == 0 {
			password := string(prefix)

			ch <- password
			return
		}
		for _, c := range chars { //for each character
			newPrefix := append(append([]byte{}, prefix...), byte(c)) //append the character to the prefix
			generate(newPrefix, length-1)                             //call the function with the new prefix and the length - 1
		}
	}
	for length := 1; length <= maxPasswordLength; length++ { //for each length
		generate([]byte{}, length)
	}
	close(ch) //close the channel
}

func main() {
	startTime := time.Now() // Capture start time

	debug.SetGCPercent(-1) // Disable GC to gain performance

	if len(os.Args) < 2 { //check if there is an argument
		log.Fatal("Please provide a file to decrypt") //if there is no argument, the program will exit
	}

	encryptedFilePath := os.Args[1]             //get the file path from the console
	data, err := os.ReadFile(encryptedFilePath) //read the file
	if err != nil {                             //check if there is an error
		log.Fatal("Error reading file: ", err)
	}

	//get the number of cores
	//not using more because it uses IO, and having more workers than cores will not increase performance
	workerCount := runtime.NumCPU()

	//create a channel with a buffer of 1000 to store the passwords
	//this ensures that the workers will not wait for the password generator to generate a password
	//this also ensures that the password generator will not wait for the workers to finish
	//no need to make it bigger cause the decryption takes more time than the password generation
	ch := make(chan string, 1000)

	//create a channel with a buffer of 1 where the workers will send the result
	resultCh := make(chan string, 1)

	//create a wait group to wait for the workers to finish
	var wg sync.WaitGroup

	//start the workers and add them to the wait group
	//this ensures that the program will not exit before the workers finish
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(data, ch, &wg, resultCh)
	}

	//start the password generator
	//no need to add it to the wait group because if the password is found before the generator finishes, the program needs to exit
	go generatePasswords(ch)

	//select statement to start the goroutine to wait for the wait group to finish
	select {
	case password := <-resultCh: //wait for the result
		fmt.Println("Password found:", password)
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

	elapsedTime := time.Since(startTime)                // Capture end time
	fmt.Printf("Program executed in %s\n", elapsedTime) // Print execution time
}
