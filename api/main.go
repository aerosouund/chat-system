package main

import (
	"chat-system/db"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var msc *db.MySQLClient
var err error

func GetApplication(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	app, err := msc.ApplicationStorage.Read(name)
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
	app, err := msc.ApplicationStorage.ReadAll()
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
	err = msc.ApplicationStorage.Write(app)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func main() {
	msc, err = db.NewMySQLClient("admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb")
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()

	// specify endpoints, handler functions and HTTP method
	router.HandleFunc("/applications", GetApplications).Methods("GET")
	router.HandleFunc("/applications", CreateApplication).Methods("POST")
	router.HandleFunc("/applications/{name}", GetApplication).Methods("GET")

	// router.HandleFunc("/applications/{name}/chats", GetChats).Methods("GET")
	// router.HandleFunc("/applications/{name}/chats/{id}", GetChat).Methods("GET")
	// router.HandleFunc("/applications/{name}/chats/{id}", CreateChat).Methods("POST")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	// start and listen to requests
	http.ListenAndServe(":8080", router)
}
