package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type simpleQueueType int

const (
	QTypeDurable   simpleQueueType = iota // Durable queue, survives server restarts
	QTypeTransient                        // Transient queue, does not survive server restarts
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType simpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	q, err := ch.QueueDeclare(
		queueName,
		queueType == QTypeDurable,
		queueType == QTypeTransient,
		queueType == QTypeTransient,
		false,
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, q, nil
}
