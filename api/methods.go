package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

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

	err = kvs.Write(context.Background(), token+"-"+newChatNumStr, "0") // check this for bugs
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
	var chatMessageCountKey = token + "-" + chatNum

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

func GetChatMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	chatNum := vars["id"]

	var chatMessageIdx = token + "-" + chatNum
	messages, err := osc.GetChatMessages(chatMessageIdx)
	fmt.Println(messages)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonData, err := json.Marshal(messages)
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonData)

}
