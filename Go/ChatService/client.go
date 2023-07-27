package main

import (
	
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/gorilla/websocket"
	"log"
	"encoding/json"
	// "sync"
)

type Client struct {
	hub 			*Hub
	Token          string
	UserID         string
	WebSocketConn  *websocket.Conn
}

type IncomingData struct {
	Receiver string `json:"Receiver"`
	Text     string `json:"Text"`
	Timestamp int64  `json:"Timestamp"`
}

type TextDB struct {
	Sender_id  string `bson:"Sender_id"`
	Timestamp int64  `bson:"Timestamp"`
	Text_id   string `bson:"Text_id"`
	Text      string `bson:"Text"`
}

func (c *Client) stopProcessing() {
	log.Printf("Closing connection with %s", c.UserID)
}

func NewClient(hub *Hub, token, userID string, conn *websocket.Conn) *Client {
	return &Client{
		hub: hub,
		Token:   token,
		UserID:    userID,
		WebSocketConn:       conn,
	}
}

func (c *Client) processOutgoing() {
	defer func() {
		c.hub.unregister <- c
		c.WebSocketConn.Close()
	}()
	
	for {
		_, message, err := c.WebSocketConn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message for %s: %s", c.UserID, err)
			break
		}

		var data IncomingData
		err = json.Unmarshal(message, &data)
		if err != nil {
			log.Println("Error unmarshaling JSON data:", err)
			continue
		}

		var text TextDB
		text = TextDB {
			Sender_id: c.UserID,
			Timestamp: data.Timestamp,
			Text_id: generate_uniqueid(),
			Text: data.Text,
		}

		// log.Printf("Sending data from %s to %s: %s\n", c.UserID, data.Receiver, data.Text)
		btext, _ := json.Marshal(text)
		if (c.hub.contains(data.Receiver)){
			go publishMessage(data.Receiver+"_q", btext)
		} else{
			go writeToDatabase(text, c.UserID, text.Sender_id)
		}
	}
}

func (c *Client) processIncoming() error{
	defer func() {
		c.WebSocketConn.Close()
	}()

	queueName := c.UserID + "_q"

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
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
		return err
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
		return err
	}
		
	for d := range msgs {
		var msg TextDB
		err := json.Unmarshal(d.Body, &msg)
		if err != nil {
			log.Println("Error unmarshaling message:", err)
			continue
		}

		err = c.WebSocketConn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error sending WebSocket message for %s: %s\n", c.UserID, err)
			btext, _ := json.Marshal(msg)

			if (c.hub.contains(c.UserID)){ 
				go publishMessage(c.UserID+"_q", btext);
			} else{
				go writeToDatabase(msg, c.UserID, msg.Sender_id)
			}
			break
		}
	}
	return nil
}

