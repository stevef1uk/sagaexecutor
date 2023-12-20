package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	dapr "github.com/dapr/go-sdk/client"

	"github.com/stevef1uk/sagaexecutor/database"
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
	App_id           string    `json:"app_id"`
	Service          string    `json:"service"`
	Token            string    `json:"token"`
	Callback_service string    `json:"callback_service"`
	Params           string    `json:"params"`
	Timeout          int       `json:"timeout"`
	Event            bool      `json:"event"`
	LogTime          time.Time `json:"logtime"`
}

type service struct {
}

var the_db *pgxpool.Pool

// Written to handle input like this. I hope there is an easier way to do this?
// input := `"app_id":sagatxs,"service":serv1,"token":abcdefg1235,"callback_service":localhost,"params":{},"Timeout":100,"TimeLogged":2023-12-16 13:09:05.837307312 +0000 UTC`
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
	the_db = database.OpenDBConnection(os.Getenv("DATABASE_URL"))
	return &service{}
}

func (service) CloseService() {
	the_db.Close()
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

func (service) SendStart(client dapr.Client, app_id string, service string, token string, callback_service string, params string, timeout int) error {
	// Base64 encode params as they should be a json string
	params = base64.StdEncoding.EncodeToString([]byte(params))
	s1 := Start_stop{App_id: app_id, Service: service, Token: token, Callback_service: callback_service, Params: params, Timeout: timeout, Event: Start, LogTime: time.Now()}
	return postMessage(client, app_id, s1)
}

func (service) SendStop(client dapr.Client, app_id string, service string, token string) error {

	s1 := Start_stop{App_id: app_id, Service: service, Callback_service: "", Token: token, Params: "", Timeout: 0, Event: Stop}
	return postMessage(client, app_id, s1)
}

func (service) GetAllLogs(client dapr.Client, app_id string, service string) {

	var log_entry Start_stop
	var mymap map[string]string
	var rawDecodedText []byte

	ret, err := database.GetStateRecords(context.Background(), the_db)
	if err != nil {
		log.Printf("Error reading state records %s", err)
		return
	}

	log.Printf("Returned %d records\n", len(ret))

	for i := 0; i < len(ret); i++ {
		res_entry := ret[i]
		//log.Printf("Basic record from DB: %v\n", res_entry)
		//log.Printf("Key = %s\n", res_entry.Key)

		/*tmp, err := strconv.Unquote(res_entry.Value)
		if err != nil {
			panic(err)
		}
		log.Printf("Called unquote ok\n")*/
		rawDecodedText, err = base64.StdEncoding.DecodeString(res_entry.Value)
		if err != nil {
			log.Printf("Base64 decode failed! %s\n", err)
			panic(err)
		}
		//log.Printf("Base64 decoded value  = %s\n", rawDecodedText)

		mymap = getMapFromString(string(rawDecodedText))

		time_logtime := mymap["logtime"]
		if time_logtime != "" {
			time_tmp := time_logtime[0:17]
			log.Printf("time_tmp = %s. time_tmp = %s\n", time_logtime, time_tmp)
			log_entry.LogTime, err = time.Parse(layout, time_tmp)
			if err != nil {
				log.Printf("Error parsing time %s\n", err)
			}
			//log.Printf("parsed time = %v\n", log_entry.LogTime)
		}

		log_entry.App_id = mymap["app_id"]
		log.Printf("App_id = %s\n", mymap["app_id"])
		log_entry.Service = mymap["service"]
		log_entry.Token = mymap["token"]
		log_entry.Timeout, _ = strconv.Atoi(mymap["timeout"])
		log_entry.Callback_service = mymap["callback_service"]

		tmp, err := strconv.Unquote(mymap["params"])
		if err != nil {
			tmp = mymap["params"]
		}
		var tmp_b []byte = make([]byte, len(tmp))
		_, _ = base64.StdEncoding.Decode(tmp_b, []byte(mymap["params"]))
		log_entry.Params = string(tmp_b)
		log.Printf("Log Entry reconstructed = %v\n", log_entry)

		elapsed := time.Since(log_entry.LogTime)
		allowed_time := log_entry.Timeout

		log.Printf("Token = %s, Elapsed value = %v, Compared value = %v\n", log_entry.Token, elapsed, allowed_time)

		if time.Duration.Seconds(elapsed) > float64(allowed_time) {
			log.Printf("Token %s, need to invoke callback %s\n", log_entry.Token, log_entry.Callback_service)

			sendCallback(client, res_entry.Key, log_entry)
		}
	}
}

func sendCallback(client dapr.Client, key string, params Start_stop) {

	data, _ := json.Marshal(params)
	content := &dapr.DataContent{
		ContentType: "application/json",
		Data:        data,
	}

	// remove the sagasubscriber|| string at the front added by Dapr

	fmt.Printf("sendCallBack invoked with key %s, params = %v\n", key, params)
	fmt.Printf("sendCallBack App_ID = %s, Method = %s\n", params.App_id, params.Callback_service)

	_, err := client.InvokeMethodWithContent(context.Background(), params.App_id, params.Callback_service, "post", content)
	if err == nil {
		// Delivered so lets delete the Start record from the Store

		err = database.Delete(context.Background(), the_db, key)
		if err == nil {
			fmt.Println("Deleted Log with key:", key)
		}
	}
}

func (service) DeleteStateEntry(key string) error {
	return database.Delete(context.Background(), the_db, key)
}

func (service) StoreStateEntry(key string, value []byte) error {
	return database.StoreState(context.Background(), the_db, key, value)
}
