package main

import (
	"chat-system/db"
	"chat-system/queue"
	"encoding/json"
	"fmt"
	"strconv"
)

var mqr *queue.RabbitMQReader
var cs db.ChatStorer

const dbString = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
const queueName = "chats"
const mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"

func main() {
	var err error

	mqr, err = queue.NewRabbitMQReader(mqttString)
	if err != nil {
		panic(err)
	}
	cs, err = db.NewChatSQLStorage(dbString)
	if err != nil {
		panic(err)
	}
	destChan := make(chan []byte)

	go mqr.Read(destChan, queueName)

	killCh := make(chan struct{})

	go func() {
		select {
		case message := <-destChan:
			var jsonMessage map[string]string
			fmt.Println("message:", string(message))

			err := json.Unmarshal(message, &jsonMessage)
			if err != nil {
				fmt.Println("invalid message, ", string(message))
			}
			token := jsonMessage["token"]
			chatNum, _ := strconv.Atoi(jsonMessage["chatNum"])
			_, err = cs.CreateChat(token, chatNum)

			if err != nil {
				fmt.Println("failed to put chat in the DB")
			}
		case <-killCh:
			mqr.CloseRecvChan()
			return
		}

	}()

	<-killCh

}
