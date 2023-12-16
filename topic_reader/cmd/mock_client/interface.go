package service

import (
	dapr "github.com/dapr/go-sdk/client"
)

type Server interface {
	SendStart(client dapr.Client, app_id string, service string, token string, callback_url string, params string, timeout int) error
	SendStop(client dapr.Client, app_id string, service string, token string) error
}
