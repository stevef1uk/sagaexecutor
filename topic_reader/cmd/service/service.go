package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	graphql "github.com/hasura/go-graphql-client"
)

const (
	PubsubComponentName     = "sagatxs"
	PubsubTopic             = "sagalogs"
	stateStoreComponentName = "sagalogs"
	Start                   = true
	Stop                    = false
	layout                  = "2006-01-02 150405"
)

type Start_stop struct {
	App_id       string    `json:"app_id"`
	Service      string    `json:"service"`
	Token        string    `json:"token"`
	Callback_url string    `json:"callback_url"`
	Params       string    `json:"params"`
	Timeout      int       `json:"timeout"`
	Event        bool      `json:"event"`
	LogTime      time.Time `json:"logtime"`
}

type State struct {
	Key   string
	Value string
}

type service struct {
}

// Written to handle input like this. I hope there is an easier way to do this?
// input := `"app_id":sagatxs,"service":serv1,"token":abcdefg1235,"callback_url":localhost,"params":{},"Timeout":100,"TimeLogged":2023-12-16 13:09:05.837307312 +0000 UTC`
func getMapFromString(input string) map[string]string {

	var m map[string]string = make(map[string]string)
	// Remove first characacter as this will be the json { othwerwise it forms part of the first key
	input2 := strings.Replace(input[1:], `"`, ``, -1)
	//fmt.Printf("input2: %s \n", input2)

	split1 := regexp.MustCompile(",").Split(input2, -1)
	for _, v := range split1 {
		split2 := regexp.MustCompile(`:`).Split(v, -1)
		//fmt.Printf("split2: %s \n", split2)
		key := ""
		for i, j := range split2 {
			//fmt.Printf("j: %s \n", j)
			if i == 0 {
				key = j
				//fmt.Printf("key: %s \n", key)
				m[key] = ""
			} else {
				m[key] = m[key] + j
				//fmt.Printf("m[%s] = %s \n", key, m[key])
			}
		}
	}
	//fmt.Printf("map = %v\n", m)
	return m
}

func NewService() Server {
	return &service{}
}

func postMessage(client dapr.Client, app_id string, s Start_stop) error {
	s_bytes, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("postMessage() failed to marshall start_stop struct %v, %s", s, err)
	}

	err = client.PublishEvent(context.Background(), PubsubComponentName, PubsubTopic, s_bytes)
	if err != nil {
		return fmt.Errorf("sendStart() failed to publish start_stop struct %q", err)
	}
	return nil
}

func (service) SendStart(client dapr.Client, app_id string, service string, token string, callback_url string, params string, timeout int) error {

	s1 := Start_stop{App_id: app_id, Service: service, Token: token, Callback_url: callback_url, Params: params, Timeout: timeout, Event: Start, LogTime: time.Now()}
	return postMessage(client, app_id, s1)
}

func (service) SendStop(client dapr.Client, app_id string, service string, token string) error {

	s1 := Start_stop{App_id: app_id, Service: service, Callback_url: "", Token: token, Params: "", Timeout: 0, Event: Stop}
	return postMessage(client, app_id, s1)
}

func (service) GetAllLogs(client dapr.Client, app_id string, service string) {

	var log_entry Start_stop
	var mymap map[string]string

	log.Println("Getting stored saga log data")
	// Need to use Hasura to query Postgres table as dapr state store query is alpha and needs some off set-up

	/*query := `{ "filter": {} }`
	ret, err := client.QueryStateAlpha1(ctx, stateStoreComponentName, query, nil)
	if err != nil {
		log.Fatalf("error reading from state store: %v", err)
	}
	*/

	ret := callHasura(app_id, service)

	log.Printf("Returnd %d records\n", len(ret))

	for i := 0; i < len(ret); i++ {
		res_entry := ret[i]
		log.Printf("Entry Key = %s\n", res_entry.Key)

		rawDecodedText, err := base64.StdEncoding.DecodeString(res_entry.Value)
		if err != nil {
			panic(err)
		}
		log.Printf("Base64 decoded value  = %s\n", rawDecodedText)

		mymap = getMapFromString(string(rawDecodedText))
		time_logtime := mymap["logtime"]
		if time_logtime != "" {
			time_tmp := time_logtime[0:17]
			log.Printf("time_tmp = %s. time_tmp = %s\n", time_logtime, time_tmp)
			log_entry.LogTime, err = time.Parse(layout, time_tmp)
			if err != nil {
				log.Printf("Error parsing time %s\n", err)
			}
			log.Printf("parsed time = %v\n", log_entry.LogTime)
		}

		log_entry.App_id = mymap["app_id"]
		log.Printf("App_id = %s\n", mymap["app_id"])
		log_entry.Service = mymap["service"]
		log_entry.Token = mymap["token"]
		log_entry.Timeout, _ = strconv.Atoi(mymap["timeout"])
		log_entry.Callback_url = mymap["callback_url"]
		log.Printf("Log Entry reconstructed = %v\n", log_entry)

		elapsed := time.Since(log_entry.LogTime)
		allowed_time := log_entry.Timeout

		log.Printf("Elapsed value = %v\n", elapsed)
		log.Printf("Compared value = %v\n", allowed_time)

		if time.Duration.Seconds(elapsed) > float64(allowed_time) {
			log.Printf("Token %s, need to invoke callback %s\n", log_entry.Token, log_entry.Callback_url)
			// remove the sagasubscriber|| string at the front added by Dapr
			key_actual := res_entry.Key[16:]
			sendCallback(client, key_actual, log_entry)
		}
	}
}

func callHasura(app_id string, service string) []State {
	url := "http://hasura.default.svc.cluster.local/v1/graphql"
	client := graphql.NewClient(url, nil)

	id := "sagasubscriber||" + app_id + service + "%"
	query := `query MyQuery { 
		      state(where: {key: {_like: ` + `"` + id + `"}}) {
				key
				value
		  }
	  }`

	//log.Println("query = " + query)
	var res struct {
		SagaLogs []State `json:"state"`
	}

	raw, err := client.ExecRaw(context.Background(), query, map[string]any{})
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(raw, &res)
	if err != nil {
		log.Printf("Error querying state store %s\n", err)
	}

	return res.SagaLogs
}

func sendCallback(client dapr.Client, key string, params Start_stop) {

	data, _ := json.Marshal(params)
	content := &dapr.DataContent{
		ContentType: "application/json",
		Data:        data,
	}

	fmt.Printf("sendCallBack invoked with key %s, params = %v\n", key, params)
	fmt.Printf("sendCallBack App_ID = %s, Method = %s\n", params.App_id, params.Callback_url)

	_, err := client.InvokeMethodWithContent(context.Background(), params.App_id, params.Callback_url, "post", content)
	if err == nil {
		// Delivered so lets delete the Start record from the Store
		err = client.DeleteState(context.Background(), stateStoreComponentName, key, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Deleted Log with key:", key)
	} else {
		fmt.Printf("Error: unable to invoke function %s for app_id %s. Error = %s\n", params.Callback_url, params.App_id, err)
	}

}
