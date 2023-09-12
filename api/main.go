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

var msc *db.MySQLClient
var err error
var mq queue.MessageQueueWriter
var as db.ApplicationStorer

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func CreateApplication(w http.ResponseWriter, r *http.Request) {
	var app map[string]string

	err := json.NewDecoder(r.Body).Decode(&app)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = as.CreateApplication(app)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func CreateChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["name"]

	if _, err := msc.ApplicationStorage.Read(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} // swap out with actual token

	chatNumber := strconv.Itoa(int(GetNextChatID()))

	createChatMessage := map[string]string{
		"applicationToken": token,
		"chatNumber":       chatNumber,
	}

	jsonString, err := json.Marshal(createChatMessage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = mq.Write("chats", string(jsonString))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := map[string]string{
		"chatNumber": chatNumber,
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonResponse))

	// write message in the queue
}

func main() {
	msc, err = db.NewMySQLClient("admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb")
	if err != nil {
		log.Fatal(err)
	}

	mq, err = queue.NewRabbitMQWriter("amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/")
	if err != nil {
		log.Fatal(err)
	}

	as, err = db.NewApplicationSQLStorage("admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb")

	router := mux.NewRouter()

	// specify endpoints, handler functions and HTTP method
	router.HandleFunc("/applications", GetApplications).Methods("GET")
	router.HandleFunc("/applications", CreateApplication).Methods("POST")
	router.HandleFunc("/applications/{name}", GetApplication).Methods("GET")

	router.HandleFunc("/applications/{name}/chats", CreateChat).Methods("POST")
	// router.HandleFunc("/applications/{name}/chats/{id}", GetChat).Methods("GET")
	// router.HandleFunc("/applications/{name}/chats/{id}", CreateChat).Methods("POST")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	// start and listen to requests
	http.ListenAndServe(":8080", router)
}
