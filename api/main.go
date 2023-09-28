package main

import (
	"chat-system/db"
	"chat-system/queue"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var (
	mq  queue.MessageQueueWriter
	as  db.ApplicationStorer
	cs  db.ChatStorer
	kvs db.KVStorage
	osc *db.OpenSearchClient
)

const (
	dbString   = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
	queueName  = "chats"
	mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"
	redisURL   = "redis://a1885c2f187ac44ba9d66f258773d630-100103902.eu-west-1.elb.amazonaws.com:6379/1"
)

func main() {
	var err error

	mq, err = queue.NewRabbitMQWriter(mqttString)
	if err != nil {
		log.Fatal(err)
	}

	as, err = db.NewApplicationSQLStorage(dbString)
	if err != nil {
		log.Fatal(err)
	}

	cs, err = db.NewChatSQLStorage(dbString)
	if err != nil {
		log.Fatal(err)
	}

	kvs, err = db.NewRedisStorage(redisURL)
	if err != nil {
		log.Fatal(err)
	}

	osc, err = db.NewOpenSearchClient("https://search-staging-z3rrlu65yks6qbepqvweu5cm7q.eu-west-1.es.amazonaws.com", "admin", "Foob@r00")
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	MakeHTTPTransport(router)
}

func MakeHTTPTransport(router *mux.Router) {
	router.HandleFunc("/applications", GetApplications).Methods("GET")
	router.HandleFunc("/applications/{name}", CreateApplication).Methods("POST")
	router.HandleFunc("/applications/{token}", GetApplication).Methods("GET")

	router.HandleFunc("/applications/{token}/chats", CreateChat).Methods("POST")
	router.HandleFunc("/applications/{token}/chats/{id}", GetChat).Methods("GET")
	router.HandleFunc("/applications/{token}/chats", GetApplicationChats).Methods("GET")
	router.HandleFunc("/applications/{token}", DeleteApplication).Methods("POST")

	router.HandleFunc("/applications/{token}/chats/{id}/messages", CreateMessage).Methods("POST")
	router.HandleFunc("/applications/{token}/chats/{id}/messages", GetChatMessages).Methods("GET")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	http.ListenAndServe(":8080", router)
}
