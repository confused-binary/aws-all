package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func check_requirements() {
	_, err := exec.LookPath("aws")
	if err != nil {
		log.Fatal("AWS CLI not found - We still need for this to be installed.")
	}

	args := os.Args
	if len(args[1:]) == 0 {
		log.Fatal("No command provided - I need to know what to run...")
	}

	_, present := os.LookupEnv("AWS_ALL_PROFILES")
	fmt.Printf("%v", present)
}

func main() {
	check_requirements()

	args := os.Args[1:]
	cmd := exec.Command("aws", args...)
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
}
