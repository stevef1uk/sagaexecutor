package service

import (
	"log"
	"testing"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

var client dapr.Client
var s Server

func setupSuite(tb testing.TB) func(tb testing.TB) {
	var err error
	log.Println("setup test suite")

	client, err = dapr.NewClient()
	if err != nil {
		panic(err)
	}

	s = NewService()

	// Return a function to teardown the test
	return func(tb testing.TB) {
		log.Println("teardown suite")
	}
}

func TestOne(t *testing.T) {

	ret := setupSuite(t)
	_ = ret
	//defer ret(t)

	err := s.SendStart(client, PubsubComponentName, "serv1", "abcdefg1234", "localhost", "{}", 100)

	if err != nil {
		t.Errorf("got %s", err)
	}

	err = s.SendStop(client, PubsubComponentName, "serv1", "abcdefg1234")

	if err != nil {
		t.Errorf("got %s", err)
	}

	time.Sleep(1 * time.Second)

}

// This simulates a service that needs to initiate a Saga Transaction to me managed
func TestTwo(t *testing.T) {
	err := s.SendStart(client, PubsubComponentName, "serv1", "abcdefg1235", "localhost", "{}", 100)

	if err != nil {
		t.Errorf("got %s", err)
	}

	client.Close()
}
