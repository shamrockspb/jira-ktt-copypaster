package main

import (
	"fmt"
	"log"
)

var globalConfig *Config
var version = "0.1"

func main() {
	
	fmt.Printf("Jira-KTT Copypaster, version %v\n\n", version)

	//Get config path and issue list to work on from CLI parameters
	configPath, jiraIssues, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	
	//Read config file
	config, err := NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	
	//Assign global config
	globalConfig = config

	//Check that all 
	CheckMandatoryConfiguration(config, jiraIssues)

	//Call main logic to copy issues from Jira to KTT
	TransferTickets(config, jiraIssues)

}

