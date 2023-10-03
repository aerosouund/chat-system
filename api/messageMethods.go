package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

func (ms *MessageServer) HandleGetChatMessages(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {

	token := routerVars["token"]
	chatNum := routerVars["id"]

	var chatMessageIdx = token + "-" + chatNum
	messages, err := ms.Opensearch.GetChatMessages(chatMessageIdx)

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonData, err := json.Marshal(messages)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, jsonData)
	}
}

func (ms *MessageServer) HandleCreateMessage(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	token := routerVars["token"]
	chatNum := routerVars["id"]

	var requestBody map[string]string
	decoder := json.NewDecoder(r.Body)

	return func(http.ResponseWriter, *http.Request) {
		if err := decoder.Decode(&requestBody); err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		ctx := context.Background()
		var chatMessageCountKey = token + "-" + chatNum

		chatMessageCountStr, err := ms.KVStore.Read(ctx, chatMessageCountKey)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		chatMessageCount, _ := strconv.Atoi(chatMessageCountStr)
		chatNumberInt, _ := strconv.Atoi(chatNum)
		newMessageCount := chatMessageCount + 1
		newMessageCountStr := strconv.Itoa(newMessageCount)

		err = ms.KVStore.Write(ctx, chatMessageCountKey, newMessageCountStr)

		err = ms.Opensearch.PutDocument(chatMessageCountKey, token, requestBody["body"], chatNumberInt, newMessageCount)
	}
}

func (ms *MessageServer) HandleSearchMessages(routerVars map[string]string, w http.ResponseWriter, r *http.Request) HttpHandlerFunc {
	queryTerm, ok := r.URL.Query()["query"]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing query term"})
	}

	token := routerVars["token"]
	chatNum := routerVars["id"]

	var chatMessageIdx = token + "-" + chatNum
	messages, err := ms.Opensearch.Search(chatMessageIdx, queryTerm[0])

	return func(http.ResponseWriter, *http.Request) {
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonData, err := json.Marshal(messages)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, jsonData)
	}
}
