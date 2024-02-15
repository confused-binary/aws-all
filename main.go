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

func worker(command <-chan []string, jobs_chan <-chan string, results_chan chan<- string, errors_chan chan<- error) {
	// for job := range jobs_chan {
	// 	// response, err := runCommand(command.Profile, command.Command)
	// 	// fmt.Println(response)
	// 	// fmt.Println(err)
	// 	fmt.Println(job)
	// }
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

	// Create Working Groups for jobs and their results
	jobs_chan := make(chan string, len(validProfiles))
	results_chan := make(chan string, len(validProfiles))
	errors_chan := make(chan string, len(validProfiles))

	// Add to Worker Pools
	poolCap := 10
	for w := 1; w <= poolCap; w++ {
		go worker(os.Args[1:], jobs_chan, results_chan, errors_chan)
	}

	// cmdDetails = []string{"sts", "get-caller-identity", "--query", "Account", "--output", "text"}
	// for w := 1; w <= poolCap; w++ {
	// 	go worker()
	// }

	// Run command against each profile - seriel for initial testing
	// results := []cmdDetails{}
	// for _, profile := range validProfiles {
	// 	var cmdDetails cmdDetails
	// 	cmdDetails.Profile = profile
	// 	accountQuery := []string{"sts", "get-caller-identity", "--query", "Account", "--output", "text"}
	// 	accountDetails, error := runCommand(profile, accountQuery)
	// 	if error != nil {

	// 	}

	// 	cmdDetails.Account, _ = strconv.Atoi(strings.Replace(accountDetails, "\n", "", -1))
	// 	cmdDetails.Results, error = runCommand(profile, os.Args[1:])
	// 	results = append(results, cmdDetails)
	// }
	// output, err := json.Marshal(&results)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(string(output))
	fmt.Println("It's done!")
}
