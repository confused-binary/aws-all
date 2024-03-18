package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Data struct {
	Profile string
	Account int
	Results map[string]interface{}
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

	profileRegex, present := os.LookupEnv("AWS_ALL")
	if !present {
		log.Fatal("\"AWS_ALL\" environment variable needs to be set so I know which profiles to run against")
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

func runCommand(profile string, command []string) map[string]interface{} {
	statement := append([]string{"--profile", profile}, command...)
	cmd := exec.Command("aws", statement...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout // Capturing standard output
	cmd.Stderr = &stderr // Capturing standard error

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Profile: %s%s", profile, stderr.String())
	}

	// Decalre empty interface
	var resultsMap map[string]interface{}

	// Unmarshal or Decode into interface
	json.Unmarshal(stdout.Bytes(), &resultsMap)

	return resultsMap
}

func workerChannels(wg *sync.WaitGroup, ch chan<- Data, profile string, argCommand []string) {
	var wgCommands sync.WaitGroup
	accountChan := make(chan int)
	resultsChan := make(chan map[string]interface{})

	// Adding 1 to the inner WaitGroup counter
	wgCommands.Add(2)

	// Start goroutines that do some work
	for i := 0; i < 1; i++ {
		go func() {
			// Run Command
			acctIdCmd := []string{"sts", "get-caller-identity"}
			resultsMap := runCommand(profile, acctIdCmd)

			// Get int for Account ID to pass to return cannel
			acctId, _ := strconv.Atoi(resultsMap["Account"].(string))
			accountChan <- acctId

			// Decrement the inner WaitGroup counter when done
			defer wgCommands.Done()
		}()

		go func() {
			// Run Command
			resultsMap := runCommand(profile, argCommand)

			// Get int for Account ID to pass to return cannel
			resultsChan <- resultsMap

			// Decrement the inner WaitGroup counter when done
			defer wgCommands.Done()
		}()

		// Send the Message through the channel
		ch <- Data{
			Profile: profile,
			Account: <-accountChan,
			Results: <-resultsChan,
		}
	}

	// Decrement the WaitGroup counter when done
	defer wg.Done()
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
	var totalData []Data
	go func() {
		for i := 0; i < len(validProfiles); i++ {
			totalData = append(totalData, <-ch)
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	b, _ := json.MarshalIndent(&totalData, "", "    ")
	fmt.Println(string(b))
}
