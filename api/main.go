package main

import (
	"github.com/gorilla/mux"
)

const (
	dbString   = "admin:pass@tcp(localhost:3306)/testdb"
	queueName  = "chats"
	mqttString = "amqp://user:pass@localhost:5672/"
	redisURL   = "redis://localhost:6379/1"
	osURL      = "localhost"
	osUser     = "admin"
	osPass     = "P@$$word"
)

func main() {
	appserver, chatserver, messageserver := initDependencies()

	router := mux.NewRouter()
	MakeHTTPTransport(appserver, chatserver, messageserver, router)
}
