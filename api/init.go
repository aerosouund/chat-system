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

	osc, err := db.NewOpenSearchClient(osURL, osUser, osPass)
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
	router.HandleFunc("/applications/{token}/chats/{id}/messages/search", ms.GetMuxVarsMiddleware(ms.HandleSearchMessages)).Methods("GET")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	http.ListenAndServe(":8080", router)
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(rw).Encode(v)
}
