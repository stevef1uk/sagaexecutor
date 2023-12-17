package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"net/http"
	"os"

	"github.com/gorilla/mux"

	dapr "github.com/dapr/go-sdk/client"
	service "sagaexecctl.sjfisher.com/service"
)

var client dapr.Client
var s service.Server

func callback(w http.ResponseWriter, r *http.Request) {
	var params service.Start_stop
	fmt.Printf("Yay callback invoked!\n")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewDecoder(r.Body).Decode(&params)

	// Here do what is necessary to recover this transaction)
	fmt.Printf("transaction callback invoked %v\n\n", params)
	json.NewEncoder(w).Encode("ok")
}

func main() {
	var err error

	/*pp_id       string    `json:"app_id"`
	  Service      string    `json:"service"`
	  Token        string    `json:"token"`
	  Callback_url string    `json:"callback_url"`
	  Params       string    `json:"params"`
	  Timeout      int       `json:"timeout"`
	  Event        bool      `json:"event"`
	  LogTime      time.Time `json:"logtime"`*/

	appPort := "6000"
	if value, ok := os.LookupEnv("APP_PORT"); ok {
		appPort = value
	}
	router := mux.NewRouter()

	log.Println("setting up handler")
	router.HandleFunc("/callback", callback).Methods("POST", "OPTIONS")
	go http.ListenAndServe(":"+appPort, router)

	log.Println("About to send a couple of messages")

	client, err = dapr.NewClient()
	if err != nil {
		panic(err)
	}

	s = service.NewService()

	log.Println("Sleepig for a bit")
	time.Sleep(10 * time.Second)

	log.Println("Finished sleeping")

	err = s.SendStart(client, "mock-client", "test2", "abcdefg1234", "callback", "{}", 60)
	if err != nil {
		log.Printf("First Publish error got %s", err)
	} else {
		log.Println("Successfully pulished first start message")
	}
	log.Println("Sleepig for a bit")
	time.Sleep(30 * time.Second)

	err = s.SendStop(client, "mock-client", "test2", "abcdefg1234")
	if err != nil {
		log.Printf("First Stop publish  error got %s", err)
	} else {
		log.Println("Successfully pulished first stop message")
	}

	s.GetAllLogs(client, "mock-client", "test2")
	//s.GetAllLogs(client, "", "")

	log.Println("Sleepig for a bit to allow time to receive any callbacks")
	time.Sleep(60 * time.Second)

	client.Close()

}
