package main

import (
	"chat-system/db"
	"chat-system/queue"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// var (
// 	mq   queue.MessageQueueWriter
// 	apps db.ApplicationStorer
// 	css  db.ChatStorer
// 	kvs  db.KVStorage
// 	osc  *db.OpenSearchClient
// )

const (
	dbString   = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
	queueName  = "chats"
	mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"
	redisURL   = "redis://a1885c2f187ac44ba9d66f258773d630-100103902.eu-west-1.elb.amazonaws.com:6379/1"
)

func main() {
	as, cs, ms := initDependencies()

	router := mux.NewRouter()
	MakeHTTPTransport(as, cs, ms, router)
}

func initDependencies() (*AppServer, *ChatServer, *MessageServer) {
	mq, err := queue.NewRabbitMQWriter(mqttString)
	if err != nil {
		log.Fatal(err)
	}

	apps, err := db.NewApplicationSQLStorage(dbString)
	if err != nil {
		log.Fatal(err)
	}

	css, err := db.NewChatSQLStorage(dbString)
	if err != nil {
		log.Fatal(err)
	}

	kvs, err := db.NewRedisStorage(redisURL)
	if err != nil {
		log.Fatal(err)
	}

	osc, err := db.NewOpenSearchClient("https://search-staging-z3rrlu65yks6qbepqvweu5cm7q.eu-west-1.es.amazonaws.com", "admin", "Foob@r00")
	if err != nil {
		log.Fatal(err)
	}

	ms := &MessageServer{
		Opensearch: osc,
		KVStore:    kvs,
		Queue:      mq,
	}
	cs := &ChatServer{
		ChatStorage: css,
		KVStore:     kvs,
		Queue:       mq,
	}

	as := &AppServer{
		ApplicationStorage: apps,
		Queue:              mq,
	}

	return as, cs, ms
}

func MakeHTTPTransport(as *AppServer, cs *ChatServer, ms *MessageServer, router *mux.Router) {

	router.HandleFunc("/applications", as.GetMuxVarsMiddleware(as.GetApplications)).Methods("GET")
	router.HandleFunc("/applications/{name}", as.GetMuxVarsMiddleware(as.CreateApplication)).Methods("POST")
	router.HandleFunc("/applications/{token}", as.GetMuxVarsMiddleware(as.GetApplication)).Methods("GET")
	router.HandleFunc("/applications/{token}", as.GetMuxVarsMiddleware(as.DeleteApplication)).Methods("POST")

	router.HandleFunc("/applications/{token}/chats", cs.GetMuxVarsMiddleware(cs.HandleCreateChat)).Methods("POST")
	router.HandleFunc("/applications/{token}/chats/{id}", cs.GetMuxVarsMiddleware(cs.HandleGetChat)).Methods("GET")
	router.HandleFunc("/applications/{token}/chats", cs.GetMuxVarsMiddleware(cs.HandleGetApplicationChats)).Methods("GET")

	router.HandleFunc("/applications/{token}/chats/{id}/messages", ms.GetMuxVarsMiddleware(ms.HandleCreateMessage)).Methods("POST")
	router.HandleFunc("/applications/{token}/chats/{id}/messages", ms.GetMuxVarsMiddleware(ms.HandleGetChatMessages)).Methods("GET")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	http.ListenAndServe(":8080", router)
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(rw).Encode(v)
}
