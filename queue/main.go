package queue

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type MessageQueueWriter interface {
	Write(string, string) error
}

type MessageQueueReader interface {
	Read(string) error
}

type RabbitMQWriter struct {
	Client *amqp.Connection
}

type RabbitMQReader struct {
	Client      *amqp.Connection
	MessageChan <-chan amqp.Delivery
}

func NewRabbitMQWriter(endpoint string) (*RabbitMQWriter, error) {
	defer logrus.Info("Started RabbitMQ client")
	connection, err := amqp.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &RabbitMQWriter{
		Client: connection,
	}, nil
}

func NewRabbitMQReader(endpoint string) (*RabbitMQReader, error) {
	defer logrus.Info("Started RabbitMQ reader")
	connection, err := amqp.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &RabbitMQReader{
		Client:      connection,
		MessageChan: make(<-chan amqp.Delivery),
	}, nil
}

func (rmqr *RabbitMQReader) Read(queueName string) error {

	ch, err := rmqr.Client.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)

	if err != nil {
		return err
	}
	rmqr.MessageChan = msgs
	return nil
}

func (rmqr *RabbitMQReader) CloseRecvChan() {
	rmqr.Client.Close()
	fmt.Println("Closed recieve channel")
}

func (rmqw *RabbitMQWriter) Write(queueName, message string) error {
	channel, err := rmqw.Client.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// declaring queue with its properties over the the channel opened
	_, err = channel.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // auto delete
		false,     // exclusive
		false,     // no wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	// publishing a message
	err = channel.Publish(
		"",        // exchange
		queueName, // key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)

	if err != nil {
		return err
	}

	logrus.Info("Successfully published message")
	return nil
}
