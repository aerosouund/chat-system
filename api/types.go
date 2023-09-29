package main

import (
	"chat-system/db"
	"chat-system/queue"
)

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
