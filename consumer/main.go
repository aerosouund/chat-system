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
	dbString   = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
	queueName  = "chats"
	mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"
	osUrl      = "https://search-staging-z3rrlu65yks6qbepqvweu5cm7q.eu-west-1.es.amazonaws.com"
	osUser     = "admin"
	osPass     = "Foob@r00"
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
