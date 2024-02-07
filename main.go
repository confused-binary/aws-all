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

func check_requirements() string {
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

func run_command(s string) cmdOut bytes.Buffer {
	statement := append([]string{"--profile", s}, os.Args[1:]...)
	cmd := exec.Command("aws", statement...)
	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	// return cmdOut
	fmt.Printf("%s out:%s\n", s, cmdOut.String())
}

func main() {
	// Ensure that required content is there
	profileRegex := check_requirements()

	// Get profiles based on AWS_ALL_PROFILES regex value
	validProfiles := getProfileNames(profileRegex)

	// Run command against each profile - seriel form for initial testing
	for _, profile := range validProfiles {
		run_command(profile)
	}
	fmt.Println("It's done!")
}
