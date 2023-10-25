package main

import (
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

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
	defer wg.Done()
	for pwd := range pwdCh {
		if decrypt(data, []byte(pwd)) {
			resCh <- pwd
			return
		}
	}
}

// this is a recursive function that generates all possible passwords
// it starts with a length of 1 and goes up to the max password length
// it has a start and end index to generate only a part of the passwords
// it sends the passwords to the channel
// it closes the channel when it finishes
func pwdGenerator(ch chan<- string, maxPwdLength int, chars string, startIndex, endIndex int) {
	var generate func([]byte, int, int, int)
	generate = func(prefix []byte, length int, idxStart int, idxEnd int) {
		if length == 0 {
			ch <- string(prefix)
			return
		}
		for idx := idxStart; idx <= idxEnd; idx++ {
			c := chars[idx]
			newPrefix := append(append([]byte{}, prefix...), byte(c))
			generate(newPrefix, length-1, 0, len(chars)-1) // Now use full range
		}
	}
	for l := 1; l <= maxPwdLength; l++ {
		generate([]byte{}, l, startIndex, endIndex)
	}
	close(ch)
}

func main() {
	start := time.Now()

	debug.SetGCPercent(-1) // Disable Garbage Collector for performance improvements

	defaultMaxPwdLength := 4
	defaultChars := "abcdefghijklmnopqrstuvwxyz"

	// Parse command-line arguments
	pathPtr := flag.String("p", "", "Provide the path of the file to decrypt.")
	maxPwdLengthPtr := flag.Int("l", defaultMaxPwdLength, "Max password length to try.")
	charsPtr := flag.String("chars", defaultChars, "Characters to use for generating the password.")
	flag.Parse()

	if *pathPtr == "" {
		log.Fatal("Provide the path of the file to decrypt using -p=filepath")
	}

	data, err := os.ReadFile(*pathPtr)
	if err != nil {
		log.Fatal("Error while parsing the file: ", err)
	}

	//get the number of cores
	//not using more because it uses IO, and having more workers than cores will not increase performance
	workers := runtime.NumCPU()

	//create a channel with a buffer of 1 where the workers will send the result
	resCh := make(chan string, 1)

	//create a wait group to wait for the workers to finish
	var wg sync.WaitGroup

	partitionSize := len(*charsPtr) / workers
	rest := len(*charsPtr) % workers

	//split the work between the workers and split also the characters between the workers
	//in this way if the password starts with a letter in the end of the alphabet, the last worker will find it faster
	for i := 0; i < workers; i++ {
		startIdx := i * partitionSize
		endIdx := (i+1)*partitionSize - 1
		if i == workers-1 {
			endIdx += rest
		}

		pwdCh := make(chan string, 1000)

		//start the workers and add them to the wait group
		//this ensures that the program will not exit before the workers finish
		wg.Add(1)
		go worker(data, &wg, pwdCh, resCh)

		//start the password generator
		//no need to add it to the wait group because if the password is found before the generator finishes, the program needs to exit
		go pwdGenerator(pwdCh, *maxPwdLengthPtr, *charsPtr, startIdx, endIdx)
	}

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
	}(): //if the wait group finishes without finding, the password is found
		fmt.Println("No password found")
	}

	endTime := time.Since(start)
	fmt.Printf("Time to execute:  %s\n", endTime)
}
