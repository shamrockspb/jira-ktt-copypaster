package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/andygrunwald/go-jira"
)

//Represents fields from Jira issue
type JiraIssue struct {
	Key               string
	Summary           string
	Estimation        string
	ParentKey         string
	ParentSummary     string
	ParentBudget      string
	ParentDescription string
}

//Represents fields from KTT issue
type KttIssue struct {
	Deadline    string //Deadline In RFC3339 format
	Description string //Long text
	ExecutorIds string //Id of assigned employee
	Name        string //Short issue name
	//ObserverIds string
	PriorityId int    //Default is p3(value - 9)
	ServiceId  int    //Value mapping between JiraIssue.ParentBudget and KttIssue.ServiceId defined in configuration  150 - IMP_22 "ServicePath":"103|566|137|150|"
	StatusId   int    //Default is Open(value - 31)
	TypeId     int    //Default is Inchcape Task(value - 1037)
	WorkflowId int    // 13 Default
	Field1130  string //Parent ticket ID in Jira(JiraIssue.ParentKey)
	Field1131  string //Estimation in hours
	Field1133  string //Start date time in RFC3339 format
	Field1211  string //Ticket ID in Jira(JiraIssue.Key)
}

//Represents statistics that will be printed at the end of processing
type TicketStatistics struct {
	TotalTickets   int
	TicketsCreated int
	Errors         int
	Duplicates     int
}

//Represents KTT http client and its authorization token
type KttClient struct {
	Client        *http.Client
	Authorization string
}

//Response body from KTT with created ticket data
type KttResponse string

//Id of created ticket in KTT
type KttTicketID string

//Id of ticket in Jira
//type JiraTicketID string



//Global Variables
var statistics TicketStatistics

//TransferTickets handles main logic for transferring tickets from Jira to KTT system
func TransferTickets(cfg *Config, issues JiraIssueList) {
	var createdTickets []KttTicketID

	//Create Jira system client
	jiraClient := getJiraClient(cfg)

	//Create KTT system client
	kttClient := getKttClient(cfg)

	//For each Jira issue in list from CLI create linked ticket in KTT
	for _, issueKey := range issues {
		statistics.TotalTickets += 1
		
		
		
		if existingIssues := searchKttIssues(issueKey, kttClient); len(existingIssues) > 0 {
			log.Printf("sd") 
		}

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
	printTransferResults(statistics, createdTickets)

}

//getJiraClient returns HTTP client for Jira system with authorization set up
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

//getKttClient returns HTTP client for KTT system with authorization token included
func getKttClient(cfg *Config) *KttClient {

	kttAuthorization := base64.StdEncoding.EncodeToString([]byte(cfg.KTT.Username + ":" + cfg.KTT.Password))

	client := &http.Client{}

	kttClient := KttClient{
		Client:        client,
		Authorization: kttAuthorization}

	return &kttClient
}

func searchKttIssues(jiraTicketID string, kttClient *KttClient) []KttTicketID {
	
	//Запрос для поиска заявок
	//https://ssc.k2consult.ru/api/task?search=SQD-622&fields=Id,ExecutorIds,Closed,Data

	//Найти заявки, удовлетворяющие критериям поиска
	//	Сконструировать запрос

	//Выделить номера заявок из строки результата
	//Вывести заявки в консоль, уточнить у пользователя - следует ли еще раз создать заявку
	//При отрицательном ответе - переходить к следующей, при пложительном - создавать 


	var kttTickets []KttTicketID
	//Prepare HTTP request
	req := constructHTTPGETRequest(jiraTicketID, kttClient)

	//Send HTTP request
	kttResponse := sendHTTPGETRequestToKTT(kttClient, req)

	if kttResponse != "" {
		//Parse response body
		kttTickets = getTicketIDs(kttResponse) 
		//log.Printf("For Jira issue %v ticket created in KTT: %v", jiraIssue.Key, kttTicketID)

	} else {
		//log.Printf("For Jira issue %v ticket not created in KTT", jiraIssue.Key)
	}

	return kttTickets


}


//getJiraIssue seachs for issue in Jira system, and returns JiraIssue struct(or error)
func getJiraIssue(issueKey string, jiraClient *jira.Client) (JiraIssue, error) {

	var jiraIssue JiraIssue

	issue, _, err := jiraClient.Issue.Get(issueKey, nil)
	if err != nil {
		log.Printf("Cannot find %v", issueKey)
		return jiraIssue, err
	}

	//Fields from ticket itself
	jiraIssue.Key = issue.Key                                         //Task key
	jiraIssue.Summary = issue.Fields.Summary                          //Task name
	jiraIssue.Estimation = issue.Fields.TimeTracking.OriginalEstimate //Estimation

	if issue.Fields.Parent != nil {
		jiraIssue.ParentKey = issue.Fields.Parent.Key //Parent task key
	}

	//Fields from parent ticket
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

//createKttTicket performs ticket creation in KTT system by provided information from Jira, and returns created ticket ID
func createKttTicket(jiraIssue JiraIssue, kttClient *KttClient, cfg *Config) KttTicketID {
	//Will store ID of created KTT ticket
	var kttTicketID string

	//Construct ticket structure
	kttIssue := constructKttIssueFromJiraIssue(jiraIssue)

	//Prepare HTTP request
	req := constructHTTPPOSTRequest(kttIssue, kttClient)

	//Send HTTP request
	kttResponse := sendHTTPPOSTRequestToKTT(kttClient, req)
	if kttResponse != "" {
		//Parse response body
		kttTicketID = getTicketID(kttResponse)
		log.Printf("For Jira issue %v ticket created in KTT: %v", jiraIssue.Key, kttTicketID)

	} else {
		log.Printf("For Jira issue %v ticket not created in KTT", jiraIssue.Key)
	}

	return KttTicketID(kttTicketID)

}

//constructHTTPPOSTRequest returns http request with body marshalled from provided kttIssue
func constructHTTPPOSTRequest(kttIssue KttIssue, kttClient *KttClient) *http.Request {
	byteKttIssue, err := json.Marshal(kttIssue)
	if err != nil {
		log.Fatal(err)
	}

	fullURL := globalConfig.KTT.URL + "api/task/"
	req, _ := http.NewRequest("POST", fullURL, bytes.NewBuffer(byteKttIssue))
	req.Header.Set("Authorization", "Basic " + kttClient.Authorization)
	req.Header.Set("Content-Type", "application/json")

	return req

}

//constructHTTPGETRequest returns http request with body marshalled from provided kttIssue
func constructHTTPGETRequest(jiraIssue string, kttClient *KttClient) *http.Request {
	
	fullURL := globalConfig.KTT.URL + "api/task/"// + "search=" + jiraIssue //+ "&fields=Id,ExecutorIds,Closed,Data"
	fmt.Println(fullURL)
	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("Authorization", "Basic " + kttClient.Authorization)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()

	q.Add("search", jiraIssue)
    //q.Add("fields", "Id,ExecutorIds,Closed,Data")
	q.Add("fields", "Id,ExecutorIds,Closed")

	req.URL.RawQuery = q.Encode()

	return req

}

//constructKttIssueFromJiraIssue transforms Jira issue to KTT format, and returns transformed message
func constructKttIssueFromJiraIssue(jiraIssue JiraIssue) KttIssue {

	var kttIssue KttIssue

	//Filled from Jira issue
	kttIssue.Description = jiraIssue.ParentDescription
	kttIssue.Name = jiraIssue.ParentSummary + "_" + jiraIssue.Summary
	kttIssue.Field1130 = jiraIssue.ParentKey
	kttIssue.Field1131 = convertEstimationToHours(jiraIssue.Estimation)
	kttIssue.Field1211 = jiraIssue.Key

	//Filled from configuration file
	kttIssue.ExecutorIds = globalConfig.KTT.TicketDefaults.ExecutorIds
	kttIssue.PriorityId = globalConfig.KTT.TicketDefaults.PriorityId
	kttIssue.ServiceId = 150 //TODO: Create mapping
	kttIssue.StatusId = globalConfig.KTT.TicketDefaults.StatusId
	kttIssue.TypeId = globalConfig.KTT.TicketDefaults.TypeId
	kttIssue.WorkflowId = globalConfig.KTT.TicketDefaults.WorkflowId

	//Calculated fields
	monday, friday := GetTicketWorkdays(globalConfig.KTT.Parameters.AddWeeks)

	kttIssue.Deadline = friday.Format(time.RFC3339) //TODO: Разобраться, почему не присваивается это поле
	kttIssue.Field1133 = monday.Format(time.RFC3339)

	/*
	if(applicationMode == "test") {
		fmt.Printf("\n\n%v\n\n", kttIssue)
	}
	*/
	return kttIssue
}

//Convert from "1w 2d 4h 45m" to "60" in hours. If minutes are present, we just round them to one hour.
func convertEstimationToHours(jiraEstimation string) string {
	//Use regexp
	totalHours := 0

	//Convert weeks to hours
	re := regexp.MustCompile(`(\d*)w`)
	submatch := re.FindStringSubmatch(jiraEstimation)

	if submatch != nil {
		if weeks, err := strconv.Atoi(submatch[1]); err == nil {
			totalHours = totalHours + weeks*40
		}
	}

	//Convert days to hours
	re = regexp.MustCompile(`(\d*)d`)
	submatch = re.FindStringSubmatch(jiraEstimation)

	if submatch != nil {
		if days, err := strconv.Atoi(submatch[1]); err == nil {
			totalHours = totalHours + days*8
		}
	}

	//Add hours to already calculated value
	re = regexp.MustCompile(`(\d*)h`)
	submatch = re.FindStringSubmatch(jiraEstimation)

	if submatch != nil {
		if hours, err := strconv.Atoi(submatch[1]); err == nil {
			totalHours = totalHours + hours
		}
	}

	//Add one hour, if some minutes are present(rounding to the up)
	re = regexp.MustCompile(`(\d*)m`)
	submatch = re.FindStringSubmatch(jiraEstimation)

	if submatch != nil {
		totalHours += 1
	}

	return strconv.Itoa(totalHours)
}

//sendHTTPPOSTRequestToKTT sends HTTP POST request to KTT via kttClient, and returns response body as string
func sendHTTPPOSTRequestToKTT(kttClient *KttClient, request *http.Request) KttResponse {
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

//sendHTTPGETRequestToKTT sends HTTP GET request to KTT via kttClient, and returns response body as string
func sendHTTPGETRequestToKTT(kttClient *KttClient, request *http.Request) KttResponse {

		response, err := kttClient.Client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()

		log.Printf("Status code %v", response.StatusCode)

		//if response.StatusCode == http.StatusOK {
			
			bodyBytes, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			//fmt.Println(bodyString)
			log.Printf("Status code %v", bodyString)
			return KttResponse(bodyString)
		//}


	//return ""
}


//getTicketID extracts ID of created KTT ticket from response body
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

//getTicketIDs extracts IDs of found KTT tickets from response body
func getTicketIDs(kttResponse KttResponse)[]KttTicketID {
	fmt.Println("In getTicketIDs")
	var ticketList []KttTicketID
	var result map[string]interface{}
	json.Unmarshal([]byte(kttResponse), &result)

	fmt.Printf("Result type %T\n",result)

	//tasks := result["Tasks"].([]map[string]interface{})
	tasks := result["Tasks"].([]interface{})
	fmt.Printf("Tasks type %T\n",tasks)
	
	for _, task := range tasks {
		fmt.Println(task.(map[string]interface{})["Id"])
		//ticketList = append(ticketList, KttTicketID(string(task.(map[string]interface{})["Id"])))	
		
	}
	



	return ticketList
/* Result
	{
		"Tasks": [
			{
				"Closed": null,
				"Data": "<field id=\"1129\" /><field id=\"1130\">SQD-620</field><field id=\"1131\">4</field><field id=\"1132\" /><field id=\"1133\">2021-06-28 09:00</field><field id=\"1134\" /><field id=\"1135\" /><field id=\"1136\" /><field id=\"1137\" /><field id=\"1138\" /><field id=\"1139\" /><field id=\"1140\" /><field id=\"1141\" /><field id=\"1142\" /><field id=\"1143\" /><field id=\"1211\">SQD-622</field>",
				"Id": 147736,
				"ExecutorIds": "5408"
			},
			{
				"Closed": null,
				"Data": "<field id=\"1129\" /><field id=\"1130\">SQD-620</field><field id=\"1131\">0</field><field id=\"1132\" /><field id=\"1133\">2021-06-28 09:00</field><field id=\"1134\" /><field id=\"1135\" /><field id=\"1136\" /><field id=\"1137\" /><field id=\"1138\" /><field id=\"1139\" /><field id=\"1140\" /><field id=\"1141\" /><field id=\"1142\" /><field id=\"1143\" /><field id=\"1211\">SQD-622</field>",
				"Id": 147734,
				"ExecutorIds": "5408"
			},
		],
		"Priorities": [],
		"Services": [],
		"Statuses": [],
		"Users": [],
		"Paginator": {
			"Count": 18,
			"Page": 1,
			"PageCount": 1,
			"PageSize": 25,
			"CountOnPage": 18,
			"HasNextPage": null
		}
	}
	*/


/*
{

    "Statuses": null,
    "Task": {
        "Deadline": null,
        "Description": "Коллеги, добрый день.",
        "ExecutorIds": "5408",
        "Hours": 0.0,
        "Id": 147710,
        "ServiceCode": "IMP_22",
        "ServiceDescription": "",
        "ServiceId": 150,
        "Field1130": "SQD-620",
        "Field1131": "0",
        "Field1133": "2021-06-28T17:00:00",
        "Field1211": "SQD-622"
    }

}


*/



}

//printTransferResults prints statistics to standard output, and also prints links to created tickets in KTT
func printTransferResults(statistics TicketStatistics, createdTickets []KttTicketID) {
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
