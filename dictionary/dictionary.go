package main

import (
	"bufio"
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
	fmt.Println(string(pwd))
	_, err := openpgp.ReadMessage(buffer, nil, prompt, nil) //try to decrypt the data, this function
	isDecryptError := err != nil
	return !isDecryptError
}

func worker(data []byte, wg *sync.WaitGroup, passwords []string, resCh chan<- string) {
	defer wg.Done()
	for _, pwd := range passwords {
		if decrypt(data, []byte(pwd)) {
			resCh <- pwd
			return
		}
	}
}

func main() {
	start := time.Now()

	debug.SetGCPercent(-1)

	// Parse command-line arguments
	pathPtr := flag.String("p", "", "Provide the path of the file to decrypt.")
	pwdFilePtr := flag.String("pwdfile", "", "Provide the path of the password file.")
	flag.Parse()
	if *pathPtr == "" {
		log.Fatal("Provide the path of the file to decrypt using -p=filepath")
	}
	if *pwdFilePtr == "" {
		log.Fatal("Provide the path of the password file using -pwdfile=passwordfile")
	}

	data, err := os.ReadFile(*pathPtr)
	if err != nil {
		log.Fatal("Error while parsing the file: ", err)
	}

	workers := runtime.NumCPU()

	resCh := make(chan string, 1)

	var wg sync.WaitGroup

	file, err := os.Open(*pwdFilePtr)
	if err != nil {
		log.Fatalf("Failed to open password file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("Failed to close password file: %v", err)
		}
	}(file)

	var passwords []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		passwords = append(passwords, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read from password file: %v", err)
	}

	// Let's distribute the passwords to the workers evenly.
	numPasswordsPerWorker := len(passwords) / workers

	for i := 0; i < workers; i++ {
		startIdx := i * numPasswordsPerWorker
		endIdx := startIdx + numPasswordsPerWorker

		// If it's the last worker or if workers don't divide the password count evenly
		// give the remaining passwords to the last worker.
		if i == workers-1 || endIdx > len(passwords) {
			endIdx = len(passwords)
		}

		wg.Add(1)
		go worker(data, &wg, passwords[startIdx:endIdx], resCh)
	}

	select {
	case pwd := <-resCh:
		fmt.Println("Password found:", pwd)
	case <-func() chan struct{} {
		ch := make(chan struct{})
		go func() {
			wg.Wait()
			close(ch)
		}()
		return ch
	}():
		fmt.Println("No password found")
	}

	endTime := time.Since(start)
	fmt.Printf("Time to execute:  %s\n", endTime)
}
