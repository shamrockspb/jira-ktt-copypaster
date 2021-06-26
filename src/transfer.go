package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/andygrunwald/go-jira"
)

type JiraIssue struct {
	Key               string
	Summary           string
	Estimation        int
	ParentKey         string
	ParentSummary     string
	ParentBudget      string
	ParentDescription string
}

type KttIssue struct {
	Deadline    string //2021-07-02T18:00:00
	Description string //Длинный текст из родительской заявки
	ExecutorIds string //Исполнитель "5408" Default(Dev Queue)
	Name        string //Наименование из родительской заявки "CRM: SQD-2438 QC56997: доработка в блоке исх. опроса для качественного опроса клиентов оценки"
	//ObserverIds string
	PriorityId int    // 9 Default
	ServiceId  int    //150 - IMP_22 "ServicePath":"103|566|137|150|",5  Нужен маппинг
	StatusId   int    //31  Открыто Default
	TypeId     int    //1037  Задача Inchcape Default
	WorkflowId int    // 13 Default
	Field1130  string //Родительская заявка "SQD-2438"
	Field1131  string //Estimation "24"
	Field1133  string //Task start date "2021-06-28T17:00:00"
	Field1211  string //ID самого тикета в Jira "SQD-3720"
}

type TicketStatistics struct {
	TotalTickets   int
	TicketsCreated int
	Errors         int
	Duplicates     int
}

type KttClient struct {
	Client        *http.Client
	Authorization string
}

type KttResponse string

type KttTicketID string

var statistics TicketStatistics

func TransferTickets(cfg *Config, issues []string) {
	var createdTickets []KttTicketID

	//Create Jira system client
	jiraClient := getJiraClient(cfg)

	//Create KTT system client
	kttClient := getKttClient(cfg)

	//For each Jira issue in list from CLI create linked ticket in KTT
	for _, issueKey := range issues {
		statistics.TotalTickets += 1
		jiraIssue, err := getJiraIssue(issueKey, jiraClient)
		if err != nil {
			statistics.Errors += 1
			log.Println(err)
			continue
		}

		ticket := createKttTicket(jiraIssue, kttClient, cfg)
		if ticket != "" {
			createdTickets = append(createdTickets, ticket)
		}

	}

	//Output results
	log.Printf("\n\nResults:")
	fmt.Printf("Tickets total: %v\n", statistics.TotalTickets)
	fmt.Printf("Tickets created: %v\n", statistics.TicketsCreated)
	fmt.Printf("Errors: %v\n", statistics.Errors)
	fmt.Printf("Duplicates: %v\n", statistics.Duplicates)

	if len(createdTickets) > 0 {
		log.Printf("\n\nCreated tickets:")

		for _, ticket := range createdTickets {
			fmt.Printf(globalConfig.KTT.URL + "Task/View/" + string(ticket) + "\n")
		}
	}

}

func getJiraClient(cfg *Config) *jira.Client {
	tp := jira.BasicAuthTransport{
		Username: cfg.Jira.Username,
		Password: cfg.Jira.Password,
	}
	jiraClient, err := jira.NewClient(tp.Client(), cfg.Jira.URL)
	if err != nil {
		log.Fatalln(err)
	}

	return jiraClient
}

func getKttClient(cfg *Config) *KttClient {

	kttAuthorization := base64.StdEncoding.EncodeToString([]byte(cfg.KTT.Username + ":" + cfg.KTT.Password))

	client := &http.Client{}

	kttClient := KttClient{
		Client:        client,
		Authorization: kttAuthorization}

	return &kttClient
}

func getJiraIssue(issueKey string, jiraClient *jira.Client) (JiraIssue, error) {

	var jiraIssue JiraIssue

	issue, _, err := jiraClient.Issue.Get(issueKey, nil)
	if err != nil {
		log.Printf("Cannot find %v", issueKey)
		return jiraIssue, err
	}

	jiraIssue.Key = issue.Key                        //Task key
	jiraIssue.Summary = issue.Fields.Summary         //Task name
	jiraIssue.Estimation = issue.Fields.TimeEstimate //Estimation
	if issue.Fields.Parent != nil {
		jiraIssue.ParentKey = issue.Fields.Parent.Key //Parent task key
	}

	if jiraIssue.ParentKey == "" {
		return jiraIssue, fmt.Errorf("issue %v does not have parent ticket, do nothing", issueKey)
	}
	issue, _, err = jiraClient.Issue.Get(jiraIssue.ParentKey, nil)
	if err != nil {
		log.Printf("Cannot find parent ticket %v", issueKey)
		return jiraIssue, err
	}

	jiraIssue.ParentSummary = issue.Fields.Summary //Parent task summary
	jiraIssue.ParentDescription = issue.Fields.Description

	//Example of issue.Fields.Unknowns["customfield_10416"]:
	//customfield_10416:map[id:12016 self:https://inchcapeglobal.atlassian.net/rest/api/2/customFieldOption/12016 value:IMP_22]
	budgetMap := issue.Fields.Unknowns["customfield_10416"] //Budget
	budgetMap2, ok := budgetMap.(map[string]interface{})
	if ok {
		jiraIssue.ParentBudget = budgetMap2["value"].(string)
	}

	return jiraIssue, nil

}

func createKttTicket(jiraIssue JiraIssue, kttClient *KttClient, cfg *Config) KttTicketID {
	//Will store ID of created KTT ticket
	var kttTicketID string

	//Construct ticket structure
	kttIssue := constructKttIssueFromJiraIssue(jiraIssue)

	//Prepare HTTP request
	req := constructHTTPRequest(kttIssue, kttClient)

	//Send HTTP request
	kttResponse := sendHTTPRequestToKTT(kttClient, req)
	if kttResponse != "" {
		//Parse response body
		kttTicketID = getTicketID(kttResponse)
		log.Printf("For Jira issue %v ticket created in KTT: %v", jiraIssue.Key, kttTicketID)

	} else {
		log.Printf("For Jira issue %v ticket not created in KTT", jiraIssue.Key)
	}

	return KttTicketID(kttTicketID)

}

func constructHTTPRequest(kttIssue KttIssue, kttClient *KttClient) *http.Request {
	byteKttIssue, err := json.Marshal(kttIssue)
	if err != nil {
		log.Fatal(err)
	}

	fullURL := globalConfig.KTT.URL + "api/task/"
	req, _ := http.NewRequest("POST", fullURL, bytes.NewBuffer(byteKttIssue))
	req.Header.Set("Authorization", "Basic "+kttClient.Authorization)
	req.Header.Set("Content-Type", "application/json")

	return req

}

func constructKttIssueFromJiraIssue(jiraIssue JiraIssue) KttIssue {

	var kttIssue KttIssue

	//Filled from Jira issue
	kttIssue.Description = jiraIssue.ParentDescription
	kttIssue.Name = jiraIssue.ParentSummary + "_" + jiraIssue.Summary
	kttIssue.Field1130 = jiraIssue.ParentKey
	kttIssue.Field1131 = "0"                   //TODO: Estimation, продумать откуда брать
	kttIssue.Field1211 = jiraIssue.Key

	//Filled from configuration file
	kttIssue.ExecutorIds = globalConfig.KTT.TicketDefaults.ExecutorIds 
	kttIssue.PriorityId = globalConfig.KTT.TicketDefaults.PriorityId  
	kttIssue.ServiceId = 150 //TODO: Create mapping
	kttIssue.StatusId = globalConfig.KTT.TicketDefaults.StatusId   
	kttIssue.TypeId =  globalConfig.KTT.TicketDefaults.TypeId 
	kttIssue.WorkflowId = globalConfig.KTT.TicketDefaults.WorkflowId 
	
	//Calculated fields
	//*Task start date(понедельник следующей недели)
	//*Срок = пятница следующей недели
	
	monday, friday := getTicketWorkdays(globalConfig.KTT.Parameters.AddWeeks)
	
	kttIssue.Deadline = friday.Format(time.RFC3339)  //TODO: Разобраться, почему не присваивается это поле
	kttIssue.Field1133 = monday.Format(time.RFC3339)
	
	log.Printf("Monday: %v", kttIssue.Field1133)
	log.Printf("Friday: %v", kttIssue.Deadline)
	
	return kttIssue
}

func sendHTTPRequestToKTT(kttClient *KttClient, request *http.Request) KttResponse {
	//Perform POST request against KTT only in "normal" mode
	if applicationMode == "normal" {

		response, err := kttClient.Client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()

		log.Printf("Status code %v", response.StatusCode)

		if response.StatusCode == http.StatusCreated {
			statistics.TicketsCreated += 1
			bodyBytes, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			//fmt.Println(bodyString)
			return KttResponse(bodyString)
		} else {
			statistics.Errors += 1
		}

	}

	if applicationMode == "test" {
		log.Println("In test mode, pass ticket creation...")
	}

	return ""
}

func getTicketID(kttResponse KttResponse) string {

	var result map[string]interface{}
	json.Unmarshal([]byte(kttResponse), &result)

	// The object stored in the "Task" key is also stored as
	// a map[string]interface{} type, and its type is asserted from
	// the interface{} type
	task := result["Task"].(map[string]interface{})

	var ticketId string

	for key, value := range task {

		if key == "Id" {
			ticketId = fmt.Sprintf("%v", int(value.(float64)))
			break
		}
	}

	return ticketId
}
