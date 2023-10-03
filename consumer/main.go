package main

import (
	"chat-system/db"
	"chat-system/queue"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

const (
	dbString   = "admin:pass@tcp(localhost:3306)/testdb"
	queueName  = "chats"
	mqttString = "amqp://user:pass@localhost:5672/"
	redisURL   = "redis://localhost:6379/1"
	osUrl      = "localhost"
	osUser     = "admin"
	osPass     = "P@$$word"
)

func main() {
	var err error

	mqr, err := queue.NewRabbitMQReader(mqttString)
	if err != nil {
		log.Fatal(err)
	}
	defer mqr.CloseRecvChan()

	cs, err := db.NewChatSQLStorage(dbString)
	if err != nil {
		log.Fatal(err)
	}

	osc, err := db.NewOpenSearchClient(osUrl, osUser, osPass)
	if err != nil {
		log.Fatal(err)
	}
	destChan := make(chan []byte)
	killCh := make(chan struct{})

	go mqr.Read(destChan, queueName)

	go func() {
		for {
			select {
			case message := <-destChan:
				var jsonMessage map[string]string
				fmt.Println("message:", string(message))

				err := json.Unmarshal(message, &jsonMessage)
				if err != nil {
					fmt.Println("invalid message, ", string(message))
				}
				token := jsonMessage["applicationToken"]
				chatNum, _ := strconv.Atoi(jsonMessage["chatNumber"])

				_, err = cs.CreateChat(token, chatNum)
				if err != nil {
					fmt.Println("failed to put chat in the DB", err)
				}

				err = osc.CreateIndex(token + "-" + jsonMessage["chatNumber"])
				if err != nil {
					fmt.Println("Failed to create elasticsearch index for chat", err)
				}
			case <-killCh:
				mqr.CloseRecvChan()
				return
			default:
				time.Sleep(time.Second * 2)
			}

		}
	}()

	<-killCh

}
