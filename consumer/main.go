package main

import (
	"chat-system/db"
	"chat-system/queue"
	"encoding/json"
	"fmt"
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

	mqr.Read(queueName)
	killCh := make(chan struct{})

	go func() {
		for {
			select {
			case message := <-mqr.MessageChan:
				var jsonMessage map[string]interface{}
				messageBytes := message.Body

				err := json.Unmarshal(messageBytes, &jsonMessage)
				if err != nil {
					fmt.Println("invalid message, ", string(messageBytes))
					continue
				}
				token := jsonMessage["token"].(string)
				chatNum := jsonMessage["chatNum"].(int)
				_, err = cs.CreateChat(token, chatNum)

				if err != nil {
					fmt.Println("failed to put chat in the DB")
					continue
				}
			case <-killCh:
				mqr.CloseRecvChan()
				return
			}
		}
	}()

}
