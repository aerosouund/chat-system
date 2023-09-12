package queue

import (
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type MessageQueueWriter interface {
	Write(queueName, message string) error
}

type RabbitMQWriter struct {
	Client *amqp.Connection
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
