// Listen to a topic and store the messages in the Dapr StateStore
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	common "github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	service "sagaexecctl.sjfisher.com/service"
)

const stateStoreComponentName = "sagalogs"

type dataElement struct {
	Data    string             `json:"data"`
	LogData service.Start_stop `json:"logdata"`
}

var sub = &common.Subscription{
	PubsubName: service.PubsubComponentName,
	Topic:      service.PubsubTopic,
	Route:      "/receivemessage",
}

var sub_client dapr.Client

func main() {
	var err error
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "7005"
	}

	sub_client, err = dapr.NewClient()
	if err != nil {
		panic(err)
	}
	//defer sub_client.Close()

	// Create the new server on appPort and add a topic listener
	s := daprd.NewService(":" + appPort)
	err = s.AddTopicEventHandler(sub, eventHandler)
	if err != nil {
		log.Fatalf("error adding topic subscription: %v", err)
	}

	log.Printf("Starting the server using port %s'n", appPort)
	// Start the server
	err = s.Start()
	if err != nil && err != http.ErrServerClosed {
		sub_client.Close()
		log.Fatalf("error listenning: %v", err)
	}
	sub_client.Close()
}

func storeMessage(client dapr.Client, m *service.Start_stop) error {
	var err error

	key := m.App_id + m.Service + m.Token

	// Only store Starts
	if m.Event == service.Start {
		t := time.Now().UTC()
		s1 := t.String()

		log_m := `{"app_id":` + m.App_id + ","
		log_m += `"service":` + m.Service + ","
		log_m += `"token":` + m.Token + ","
		log_m += `"callback_url":` + m.Callback_url + ","
		log_m += `"params":` + m.Params + ","
		log_m += `"event": true` + ","
		log_m += `"timeout":` + strconv.Itoa(m.Timeout) + ","
		log_m += `"logtime":` + s1 + "}"

		log.Printf("Storing key = %s, data = %s\n", key, log_m)

		// Save state into the state store
		err = client.SaveState(context.Background(), stateStoreComponentName, key, []byte(log_m), nil)
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("Saved Log:", string(log_m))
		}
	} else { // Stop means we delete the corresponding Start entry
		// Delete state from the state store

		/*_, err = client.GetState(context.Background(), stateStoreComponentName, key, nil)
		if err != nil {
			fmt.Printf("Error can't find key! %s/n", err)
		}*/

		err = client.DeleteState(context.Background(), stateStoreComponentName, key, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Deleted Log with key:", key)
		/* Double check!
		_, err = client.GetState(context.Background(), stateStoreComponentName, key, nil)
		if err == nil {
			fmt.Println("Error: Key still exists after deletion attemp!")
		}*/

	}

	//client.QueryStateAlpha1()
	return err
}

func eventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	var message service.Start_stop

	var m map[string]interface{} = e.Data.(map[string]interface{})

	fmt.Println("Subscriber received:", e.Data)

	message.App_id = m["app_id"].(string)
	message.Service = m["service"].(string)
	message.Token = m["token"].(string)
	message.Callback_url = m["callback_url"].(string)
	message.Params = m["params"].(string)
	message.Timeout = int(m["timeout"].(float64))
	message.Event = m["event"].(bool)
	message.LogTime, _ = time.Parse(time.RFC3339Nano, m["logtime"].(string))

	fmt.Println("Message created", message)

	err = storeMessage(sub_client, &message)
	if err != nil {
		log.Fatalf("Unable to store message %s", err)
	}

	return false, err
}
