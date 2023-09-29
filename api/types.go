package main

import (
	"chat-system/db"
	"chat-system/queue"
	"net/http"
)

type ExecFunc func(map[string]string, http.ResponseWriter, *http.Request) HttpHandlerFunc
type HttpHandlerFunc func(http.ResponseWriter, *http.Request)

type AppServer struct {
	Queue              queue.MessageQueueWriter
	ApplicationStorage db.ApplicationStorer
}

type ChatServer struct {
	Queue       queue.MessageQueueWriter
	KVStore     db.KVStorage
	ChatStorage db.ChatStorer
}

type MessageServer struct {
	Queue      queue.MessageQueueWriter
	KVStore    db.KVStorage
	Opensearch *db.OpenSearchClient
}
