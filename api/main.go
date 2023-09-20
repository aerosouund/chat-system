package main

import (
	"chat-system/db"
	"chat-system/queue"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var mq queue.MessageQueueWriter
var as db.ApplicationStorer
var cs db.ChatStorer
var kvs db.KVStorage

const dbString = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
const queueName = "chats"
const mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"
const redisURL = "a1885c2f187ac44ba9d66f258773d630-1261174797.eu-west-1.elb.amazonaws.com"

var NextChatID atomic.Uint64

func GetNextChatID() uint64 {
	return NextChatID.Add(1)
}

func GetApplication(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	app, err := as.GetApplication(name)
	if err != nil {
		logrus.Info("Error in fetching application", err)
	}
	jsonData, err := json.Marshal(app)
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonData)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func GetApplications(w http.ResponseWriter, r *http.Request) {
	app, err := as.GetAll()
	if err != nil {
		logrus.Info("Error in fetching applications", err)
	}
	jsonData, err := json.Marshal(app)
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonData)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

}

func CreateApplication(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	app, err := as.CreateApplication(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonData, err := json.Marshal(app)
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonData)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func GetChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	chatNum, err := strconv.Atoi(vars["token"])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	chat, err := cs.GetChat(token, chatNum)
	jsonData, err := json.Marshal(chat)
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonData)
}

func GetApplicationChats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	chats, err := cs.GetAllAppChats(token)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonData, err := json.Marshal(chats)
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonData)
}

func CreateChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	if _, err := as.GetApplication(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} // swap out with actual token

	// try to read the cache and if it doesnt exist create a cache key with val 0
	// if exists the chat number is the value

	chatNumber := strconv.Itoa(int(GetNextChatID()))

	createChatMessage := map[string]string{
		"applicationToken": token,
		"chatNumber":       chatNumber,
	}

	jsonString, err := json.Marshal(createChatMessage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = mq.Write(queueName, string(jsonString))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := map[string]string{
		"chatNumber": chatNumber,
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Fprintf(w, string(jsonResponse))
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(rw).Encode(v)
}

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

	router := mux.NewRouter()
	MakeHTTPTransport(router)

	// specify endpoints, handler functions and HTTP method

}

func MakeHTTPTransport(router *mux.Router) {
	router.HandleFunc("/applications", GetApplications).Methods("GET")
	router.HandleFunc("/applications/{name}", CreateApplication).Methods("POST")
	router.HandleFunc("/applications/{name}", GetApplication).Methods("GET")

	router.HandleFunc("/applications/{token}/chats", CreateChat).Methods("POST")
	router.HandleFunc("/applications/{name}/chats/{id}", GetChat).Methods("GET")
	router.HandleFunc("/applications/{name}/chats", GetApplicationChats).Methods("GET")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	// start and listen to requests
	http.ListenAndServe(":8080", router)
}
