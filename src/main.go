package main

import (
	"log"
)

func main() {

	//TODO: Вынести issues в настройки(command-line)
	/*
		issueKey := "SQD-622"
		var issues []string
		issues = append(issues, issueKey)
	*/
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
	
	//Check that all 
	CheckMandatoryConfiguration(config, jiraIssues)

	//Call main logic to copy issues from Jira to KTT
	TransferTicket(config, jiraIssues)

}

/*
func getKttTicket(jiraIssue JiraIssue) (ticketId string, err error) {

	kttAuthorization := base64.StdEncoding.EncodeToString([]byte(kttUsername + ":" + kttPassword))

	client := &http.Client{}
	fullURL := kttURL + "api/task/" + "147430"
	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("Authorization", "Basic "+kttAuthorization)
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()


	if res.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		fmt.Println(bodyString)
	}


	return "123456", nil
}
*/
