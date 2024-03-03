package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

type Data struct {
	Profile string
	Account int
	// Command []string
	Results string
}

type Results struct {
	Details string
}

func checkRequirements() string {
	_, err := exec.LookPath("aws")
	if err != nil {
		log.Fatal("AWS CLI not found - We still need for this to be installed.")
	}

	args := os.Args
	if len(args[1:]) == 0 {
		log.Fatal("No command provided - I need to know what to run...")
	}

	profileRegex, present := os.LookupEnv("AWS_ALL_PROFILES")
	if !present {
		log.Fatal("\"AWS_ALL_PROFILES\" environment variable needs to be set so I know which profiles to run against")
	}

	return profileRegex
}

func getProfileNames(profileRegex string) []string {
	result, err := exec.Command("aws", "configure", "list-profiles").Output()
	if err != nil {
		log.Panicf("%v", err)
	}

	profiles := strings.FieldsFunc(string(result), func(r rune) bool {
		return r == '\n'
	})

	var validProfiles []string
	for _, profile := range profiles {
		match, err := regexp.MatchString(profileRegex, profile)
		if err != nil {
			println(err)
		}
		if match {
			validProfiles = append(validProfiles, profile)
		}
	}

	return validProfiles
}

// func worker(profile string, command []string, results chan<- string) {
// 	cmdOutString, err := runCommand(profile, command)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	fmt.Println(cmdOutString)
// 	results <- cmdOutString
// }

func runCommand(profile string, command []string) string {
	statement := append([]string{"--profile", profile}, command...)
	cmd := exec.Command("aws", statement...)

	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = os.Stderr

	fmt.Println(cmd.Err)

	cmdOutString := cmdOut.String()
	return cmdOutString
}

func workerChannels(wg *sync.WaitGroup, ch chan<- Data, profile string, command []string) {
	defer wg.Done() // Decrement the WaitGroup counter when done

	var wgCommands sync.WaitGroup

	// Adding 1 to the inner WaitGroup counter
	wgCommands.Add(2)

	// Start goroutines that do some work
	for i := 0; i < 1; i++ {
		accountChan := make(chan int)

		// Create a Message struct with the data and sender identifier
		go func() {
			defer wgCommands.Done() // Decrement the inner WaitGroup counter when done
			profileCmd := []string{"sts", "get-caller-identity"}
			result := runCommand(profile, profileCmd)

			var env Results
			buf := []byte(result)
			if err := json.Unmarshal(buf, &env); err != nil {
				log.Fatal(err)
			}
			account := 123

			// var resultMap map[string]any
			// var resultMap map[string]interface{}
			// json.Unmarshal([]byte(result), &resultMap)
			// account, _ := extractString(data, "person.address.city")
			// var acctString any
			// acctString = resultMap["Account"]
			// account, _ := strconv.Atoi(acctString)
			accountChan <- account
		}()

		resultsChan := make(chan string)
		go func() {
			defer wgCommands.Done()
			resultsChan <- "done"
		}()

		// Send the Message through the channel
		ch <- Data{
			Profile: profile,
			Account: <-accountChan,
			Results: <-resultsChan,
		}
	}
}

func main() {
	// Ensure that required content is there
	profileRegex := checkRequirements()

	// Get profiles based on AWS_ALL_PROFILES regex value
	validProfiles := getProfileNames(profileRegex)

	// Combine arguments to single string with spacces
	argCommand := os.Args[1:]

	// Create a channel to communicate Data structs
	ch := make(chan Data)

	// Create a WaitGroup
	var wg sync.WaitGroup

	// Add  to the WaitGroup counter to wait for goroutines
	wg.Add(len(validProfiles))

	// Start goroutine to send data through the channel with sender identifiers
	for _, profile := range validProfiles {
		go workerChannels(&wg, ch, profile, argCommand)
	}

	// Start a goroutine to receive and handle data from the channel
	go func() {
		defer wg.Done() // Decrement the WaitGroup counter when done
		for i := 0; i < len(validProfiles); i++ {
			data := <-ch
			fmt.Printf("Received data from sender %s: %v\n", data.Profile, data.Account)
			fmt.Printf("Results: %s\n", data.Results)
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// // Setup waitgroups
	// var wg sync.WaitGroup

	// // Setup channels for concurrency
	// results := make(chan string, len(validProfiles))

	// // Get Account number
	// for n, profile := range validProfiles {
	// 	wg.add(n)

	// }

	// // Pass commands to worker funciton
	// for n, profile := range validProfiles {
	// 	wg.Add(n)
	// 	profile := profile
	// 	go func() {
	// 		defer wg.Done()
	// 		data := dataStorage{
	// 			Account: ,
	// 		}
	// 		worker(profile, argCommand, results)
	// 	}()
	// }
	// wg.Wait()

	// // Get results
	// for i := 1; i <= len(validProfiles); i++ {
	// 	<-results
	// }

	fmt.Println("It's done!")
}
