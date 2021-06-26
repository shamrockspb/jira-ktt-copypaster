package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

//Mode
type ApplicationMode string

//Path to configuration file
type ConfigPath string

//List of Jira issues to process
type JiraIssueList []string

// Config struct for copypaster config
type Config struct {
	KTT struct {
		//URL to KTT server
		URL string `yaml:"url"`
		//Username for connecting to KTT
		Username string `yaml:"username"`
		//Password for connecting to KTT
		Password string `yaml:"password"`

		TicketDefaults struct {
			ExecutorIds string `yaml:"ExecutorIds"`
			PriorityId int     `yaml:"PriorityId"`
			StatusId int     	`yaml:"StatusId"`
			TypeId int     `yaml:"TypeId"`
			WorkflowId int     `yaml:"WorkflowId"`

		} `yaml:"TicketDefaults"`

		Parameters struct {
			AddWeeks int `yaml:"AddWeeks"`

		} `yaml:"Parameters"`


	} `yaml:"KTT"`

	Jira struct {
		//URL to Jira server
		URL string `yaml:"url"`
		//Username for connecting to Jira
		Username string `yaml:"username"`
		//Password for connecting to Jira
		Password string `yaml:"password"`
	} `yaml:"Jira"`
}
  

//Current application mode(default=normal)
var applicationMode ApplicationMode



//Possible application modes(constant array)
var applicationModes = [...]ApplicationMode {"normal", "test"}

//Check mandatory fields in configuration and in CLI parameters
func CheckMandatoryConfiguration(cfg *Config, jiraIssues JiraIssueList) {
	fatalError := false

	if cfg.KTT.Username == "" {
		log.Println("No login is specified for Intraservice")
		fatalError = true
	}
	if cfg.KTT.Password == "" {
		log.Println("No password is specified for Intraservice")
		fatalError = true
	}
	if cfg.KTT.URL == "" {
		log.Println("No URL is specified for Intraservice")
		fatalError = true
	}
	if cfg.Jira.Username == "" {
		log.Println("No login is specified for Intraservice")
		fatalError = true
	}
	if cfg.Jira.Password == "" {
		log.Println("No password is specified for Intraservice")
		fatalError = true
	}
	if cfg.Jira.URL == "" {
		log.Println("No URL is specified for Intraservice")
		fatalError = true
	}

	if len(jiraIssues) == 0 {
		log.Println("No Jira issues provided")
		fatalError = true
	}

	if fatalError {
		log.Fatalln("Please correct errors above")
	}

}

// NewConfig returns a new decoded Config struct
func NewConfig(configPath ConfigPath) (*Config, error) {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(string(configPath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

// ParseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func ParseFlags() (ConfigPath, JiraIssueList, error) {
	// String that contains the configured configuration path
	var configPathStr string
	var appModeStr string
	

	// Set up a CLI flag called "-config" to allow users
	// to supply the configuration file
	flag.StringVar(&configPathStr, "config", "~/.config/jira-ktt-copypaster/config.yaml", "path to config file")

	flag.StringVar(&appModeStr, "mode", "normal", "application mode")

	
	
	// Actually parse the flags
	flag.Parse()

	//APPLICATION MODE
	applicationMode = ApplicationMode(appModeStr)
	if err := validateApplicationMode(applicationMode); err != nil {
		return "", nil, err
	}
	
	//JIRA TICKETS 
	//All the rest(list og jira issues separated by )
	tail := flag.Args()
	log.Printf("List of Jira tickets to process: %+q\n", tail)

	//PATH
	// Validate the path first
	if err := validateConfigPath(configPathStr); err != nil {
		return "", nil, err
	}

	configPath := ConfigPath(configPathStr)
	jiraIssueList := JiraIssueList(tail)

	// Return the configuration path
	return configPath, jiraIssueList, nil
}

// ValidateConfigPath just makes sure, that the path provided is a file,
// that can be read
func validateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

func validateApplicationMode(appMode ApplicationMode) error {

	for _, mode := range applicationModes {
        if mode == appMode {
            return nil
        }
    }
	
    return fmt.Errorf("invalid application mode. Please select one of: %+q",applicationModes)
}