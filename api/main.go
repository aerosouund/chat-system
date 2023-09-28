package main

import (
	"chat-system/db"
	"chat-system/queue"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"

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

const dbString = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
const queueName = "chats"
const mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"
const redisURL = "redis://a1885c2f187ac44ba9d66f258773d630-100103902.eu-west-1.elb.amazonaws.com:6379/1"

var NextChatID atomic.Uint64

func GetNextChatID() uint64 {
	return NextChatID.Add(1)
}

func GetApplication(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["token"]
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

func DeleteApplication(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	err := as.DeleteApplication(token)
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
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx := context.Background()

	chatNumStr, err := kvs.Read(ctx, token)
	if err != nil {
		err = kvs.Write(ctx, token, "0")
		chatNumStr = "0"
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	chatNum, _ := strconv.Atoi(chatNumStr)
	newChatNum := chatNum + 1
	newChatNumStr := strconv.Itoa(newChatNum)
	err = kvs.Write(ctx, token, newChatNumStr)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	createChatMessage := map[string]string{
		"applicationToken": token,
		"chatNumber":       newChatNumStr,
	}

	jsonString, err := json.Marshal(createChatMessage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = kvs.Write(context.Background(), token+"+"+newChatNumStr, "0") // check this for bugs
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = mq.Write(queueName, string(jsonString))
	// create an elasticsearch index in the consumer, only create the message if the cache key exists
	// to avoid message loss in case you're writing to a chat not yet created and to decrease the cost of the create message call
	// and not have it do many checks

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := map[string]string{
		"chatNumber": newChatNumStr,
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

func CreateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	chatNum := vars["id"]

	var requestBody map[string]string
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&requestBody); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx := context.Background()
	var chatMessageCountKey = token + "+" + chatNum

	chatMessageCountStr, err := kvs.Read(ctx, chatMessageCountKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	chatMessageCount, _ := strconv.Atoi(chatMessageCountStr)
	chatNumberInt, _ := strconv.Atoi(chatNum)
	newMessageCount := chatMessageCount + 1
	newMessageCountStr := strconv.Itoa(newMessageCount)

	err = kvs.Write(ctx, chatMessageCountKey, newMessageCountStr)

	err = osc.PutDocument(chatMessageCountKey, token, requestBody["body"], chatNumberInt, newMessageCount)

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

	osc, err = db.NewOpenSearchClient("https://search-staging-z3rrlu65yks6qbepqvweu5cm7q.eu-west-1.es.amazonaws.com", "admin", "Foob@r00")
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
	router.HandleFunc("/applications/{token}", GetApplication).Methods("GET")

	router.HandleFunc("/applications/{token}/chats", CreateChat).Methods("POST")
	router.HandleFunc("/applications/{token}/chats/{id}", GetChat).Methods("GET")
	router.HandleFunc("/applications/{token}/chats", GetApplicationChats).Methods("GET")
	router.HandleFunc("/applications/{token}", DeleteApplication).Methods("POST")

	router.HandleFunc("/applications/{token}/chats/{id}/messages", CreateMessage).Methods("post")

	http.Handle("/", router)
	logrus.Info("Api server initialized")

	// start and listen to requests
	http.ListenAndServe(":8080", router)
}
