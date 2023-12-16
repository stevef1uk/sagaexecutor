package main

import (
	"log"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	service "sagaexecctl.sjfisher.com/service"
)

var client dapr.Client
var s service.Server

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

	log.Println("About to send a couple of messages")

	client, err = dapr.NewClient()
	if err != nil {
		panic(err)
	}

	s = service.NewService()

	log.Println("Sleepig for a bit")
	time.Sleep(10 * time.Second)

	log.Println("Finished sleeping")

	err = s.SendStart(client, service.PubsubComponentName, "serv1", "abcdefg1234", "localhost", "{}", 10)
	if err != nil {
		log.Printf("First Publish error got %s", err)
	} else {
		log.Println("Successfully pulished first start message")
	}

	err = s.SendStop(client, service.PubsubComponentName, "serv1", "abcdefg1234")
	if err != nil {
		log.Printf("First Stop publish  error got %s", err)
	} else {
		log.Println("Successfully pulished first stop message")
	}

	// Now try the read events test
	err = s.SendStart(client, service.PubsubComponentName, "serv1", "abcdefg1237", "localhost", "{}", 120)
	if err != nil {
		log.Printf("First Publish error got %s", err)
	} else {
		log.Println("Successfully pulished second start message")
	}

	s.GetAllLogs(client, service.PubsubComponentName, "serv1")
	s.GetAllLogs(client, "", "")

	client.Close()

}
