package main

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"fmt"
	// "encoding/json"[]
)

const rabbitMQURL = "amqp://guest:guest@localhost:5672/"

var rabbitMQConn *amqp.Connection
var rabbitMQChannel *amqp.Channel

func initRabbitMQ() error {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return err
	}
	rabbitMQConn = conn

	channel, err := rabbitMQConn.Channel()
	if err != nil {
		return err
	}
	rabbitMQChannel = channel

	// You can also declare exchanges, queues, and bindings here if required.

	log.Println("RabbitMQ initialized successfully")
	return nil
}

// Close RabbitMQ connections when the application exits.
func closeRabbitMQ() {
	if rabbitMQChannel != nil {
		rabbitMQChannel.Close()
	}
	if rabbitMQConn != nil {
		rabbitMQConn.Close()
	}
}

// publishMessage publishes a message to the specified RabbitMQ queue.
func publishMessage(queueName string, messageData []byte) error {
	if rabbitMQChannel == nil {
		return fmt.Errorf("RabbitMQ channel is not initialized")
	}

	// Declare the queue (if it does not exist)
	_, err := rabbitMQChannel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Publish the message to the queue
	err = rabbitMQChannel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        messageData,
	})
	if err != nil {
		return err
	}

	// log.Printf("Message published to queue %s", queueName)
	return nil
}

func readFromQueue(queueName string) (<-chan amqp.Delivery, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, // name
		true,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return nil, err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, err
	}

	return msgs, nil

	// messageChan := make(chan Message)

	// go func() {
	// 	defer close(messageChan)
	// 	for d := range msgs {
	// 		var msg Message
	// 		err := json.Unmarshal(d.Body, &msg)
	// 		if err != nil {
	// 			log.Println("Error unmarshaling message:", err)
	// 			continue
	// 		}
	// 		messageChan <- msg
	// 	}
	// }()

	// return messageChan, nil
}

// // readFromQueue starts a goroutine to read messages from the specified RabbitMQ queue.
// func readFromQueue(queueName string) chan []byte {
// 	messageChan := make(chan []byte)
// 	go func() {
// 		if rabbitMQChannel == nil {
// 			log.Println("RabbitMQ channel is not initialized")
// 			close(messageChan)
// 			return
// 		}

// 		_, err := rabbitMQChannel.QueueDeclare(queueName, true, false, false, false, nil)
// 		if err != nil {
// 			log.Println("Error declaring queue:", err)
// 			close(messageChan)
// 			return
// 		}

// 		deliveries, err := rabbitMQChannel.Consume(queueName, "", false, false, false, false, nil)
// 		if err != nil {
// 			log.Println("Error consuming from queue:", err)
// 			close(messageChan)
// 			return
// 		}

// 		for delivery := range deliveries {
// 			messageChan <- delivery.Body
// 			delivery.Ack(false)	
// 		}
// 	}()

// 	return messageChan
// }