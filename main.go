package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type cmdDetails struct {
	Profile string
	Account int
	Command []string
	Results string
	Error   error
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

func worker(profile string, command []string, results chan<- string) {
	cmdOutString, err := runCommand(profile, command)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(cmdOutString)
	results <- cmdOutString
}

func runCommand(profile string, command []string) (string, error) {
	statement := append([]string{"--profile", profile}, command...)
	cmd := exec.Command("aws", statement...)

	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	cmdOutString := cmdOut.String()
	return cmdOutString, nil
}

func main() {
	// Ensure that required content is there
	profileRegex := checkRequirements()

	// Get profiles based on AWS_ALL_PROFILES regex value
	validProfiles := getProfileNames(profileRegex)

	// Combine arguments to single string with spacces
	argCommand := os.Args[1:]

	// Setup channels for concurrency
	results := make(chan string, len(validProfiles))

	// Pass commands to worker funciton
	for _, profile := range validProfiles {
		go worker(profile, argCommand, results)
	}

	// Print results
	for i := 1; i <= len(validProfiles); i++ {
		<-results
	}

	fmt.Println("It's done!")
}
