package pubsub

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type qType int

const (
	QTypeDurable   qType = iota // Durable queue, survives server restarts
	QTypeTransient              // Transient queue, does not survive server restarts
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType qType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		queueType == QTypeDurable,
		queueType == QTypeTransient,
		queueType == QTypeTransient,
		false,
		amqp.Table{
			"x-dead-letter-exchange": routing.ExchangePerilDLX,
		},
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}

	return ch, q, nil
}

type Acktype int

const (
	Ack Acktype = iota
	NackDiscard
	NackRequeue
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType qType,
	handler func(T) Acktype,
) error {
	ch, q, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}

	go func() {
		defer ch.Close()
		for msg := range msgs {
			var body T
			if err := json.Unmarshal(msg.Body, &body); err != nil {
				log.Printf("error unmarshalling message: %v\n", err)
				continue
			}
			switch handler(body) {
			case Ack:
				msg.Ack(false)
				log.Println("Ack")
			case NackDiscard:
				msg.Nack(false, false)
				log.Println("NackDiscard")
			case NackRequeue:
				msg.Nack(false, true)
				log.Println("NackRequeue")
			}
		}
	}()

	return nil
}
