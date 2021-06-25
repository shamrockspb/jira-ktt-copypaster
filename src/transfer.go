package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

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

func TransferTicket(cfg *Config, issues []string) {

	jiraClient := getJiraClient(cfg)

	for _, issueKey := range issues {

		jiraIssue, err := getJiraIssue(issueKey, jiraClient)
		if err != nil {
			log.Println(err)
			continue
		}

		ticketId, err := createKttTicket(jiraIssue, cfg)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(jiraIssue)

		fmt.Println(ticketId)
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
	jiraIssue.ParentKey = issue.Fields.Parent.Key    //Parent task key

	issue, _, err = jiraClient.Issue.Get(jiraIssue.ParentKey, nil)
	if err != nil {
		log.Fatalln(err)
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

func createKttTicket(jiraIssue JiraIssue, cfg *Config) (ticketId string, err error) {

	//Заполнить по дефолту
	//*Статус = Открыто
	//*Task start date(понедельник следующей недели)
	//*Срок = пятница следующей недели
	//*Ответственный = DevQueue

	kttAuthorization := base64.StdEncoding.EncodeToString([]byte(cfg.KTT.Username + ":" + cfg.KTT.Password))

	var kttIssue KttIssue

	kttIssue.Deadline = "2021-07-02T18:00:00" //TODO: Сделать динамический расчет
	kttIssue.Description = jiraIssue.ParentDescription
	kttIssue.ExecutorIds = "5408" //Default
	kttIssue.Name = jiraIssue.ParentSummary
	kttIssue.PriorityId = 9  //Default
	kttIssue.ServiceId = 150 //Create mapping
	kttIssue.StatusId = 31   //Default
	kttIssue.TypeId = 1037   //Задача Inchcape Default
	kttIssue.WorkflowId = 13 //Default
	kttIssue.Field1130 = jiraIssue.ParentKey
	kttIssue.Field1131 = "0"                   //Estimation, продумать откуда брать
	kttIssue.Field1133 = "2021-06-28T17:00:00" //Task start date Сделать динамический расчет
	kttIssue.Field1211 = jiraIssue.Key

	byteKttIssue, err := json.Marshal(kttIssue)

	//var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)

	client := &http.Client{}
	fullURL := cfg.KTT.URL + "api/task/"
	req, _ := http.NewRequest("POST", fullURL, bytes.NewBuffer(byteKttIssue))
	req.Header.Set("Authorization", "Basic "+kttAuthorization)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	fmt.Println("Status code ")
	fmt.Print(res.StatusCode)
	if res.StatusCode == http.StatusCreated {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		fmt.Println(bodyString)
	}

	//TODO: Correct return value
	return "123456", nil
}
