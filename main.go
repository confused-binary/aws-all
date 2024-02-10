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
)

type cmdDetails struct {
	Profile string
	Account int
	Results string
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
	if present == false {
		log.Fatal("\"AWS_ALL_PROFILES\" environment variable needs to be set so I know which profiles to run against")
	}

	return profileRegex
}

func getProfileNames(profileRegex string) []string {
	result, err := exec.Command("aws", "configure", "list-profiles").Output()
	if err != nil {
		log.Panicf("%v", err)
	}

	ds := strings.FieldsFunc(string(result), func(r rune) bool {
		return r == '\n'
	})

	var validProfiles []string
	for _, d := range ds {
		match, _ := regexp.MatchString(profileRegex, d)
		if match == true {
			validProfiles = append(validProfiles, d)
		}
	}

	return validProfiles
}

func runCommand(profile string, command []string) string {
	statement := append([]string{"--profile", profile}, command...)
	cmd := exec.Command("aws", statement...)

	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	cmdOutString := cmdOut.String()
	return cmdOutString
}

func main() {
	// Ensure that required content is there
	profileRegex := checkRequirements()

	// Get profiles based on AWS_ALL_PROFILES regex value
	validProfiles := getProfileNames(profileRegex)

	// Run command against each profile - seriel for initial testing
	results := []cmdDetails{}
	for _, profile := range validProfiles {
		var cmdDetails cmdDetails
		cmdDetails.Profile = profile
		accountQuery := []string{"sts", "get-caller-identity", "--query", "Account", "--output", "text"}
		cmdDetails.Account, _ = strconv.Atoi(strings.Replace(runCommand(profile, accountQuery), "\n", "", -1))
		cmdDetails.Results = runCommand(profile, os.Args[1:])
		results = append(results, cmdDetails)
	}
	output, err := json.Marshal(&results)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(output))
	fmt.Println("It's done!")
}
