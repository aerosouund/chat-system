package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

func (cs *ChatServer) HandleGetChat(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	token := routerVars["token"]
	chatNum, err := strconv.Atoi(routerVars["token"])
	chat, err := cs.ChatStorage.GetChat(token, chatNum)

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonData, err := json.Marshal(chat)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, jsonData)
	}
}

func (cs *ChatServer) HandleGetApplicationChats(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	token := routerVars["token"]
	chats, err := cs.ChatStorage.GetAllAppChats(token)

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonData, err := json.Marshal(chats)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, jsonData)
	}
}

func (cs *ChatServer) HandleCreateChat(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	token := routerVars["token"]

	return func(http.ResponseWriter, *http.Request) {
		// handle creation of a chat on a non existing application

		ctx := context.Background()

		chatNumStr, err := cs.KVStore.Read(ctx, token)
		if err != nil {
			err = cs.KVStore.Write(ctx, token, "0")
			chatNumStr = "0"
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		chatNum, _ := strconv.Atoi(chatNumStr)
		newChatNum := chatNum + 1
		newChatNumStr := strconv.Itoa(newChatNum)
		err = cs.KVStore.Write(ctx, token, newChatNumStr)

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

		err = cs.KVStore.Write(context.Background(), token+"-"+newChatNumStr, "0")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		err = cs.Queue.Write(queueName, string(jsonString))
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

		writeJSON(w, http.StatusOK, jsonResponse)
	}
}
