package main

import (
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitURL = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	log.Println("Peril game server connected to RabbitMQ!")

	pubCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not open a channel: %v", err)
	}

	_, queue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.QTypeDurable,
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	log.Printf("Queue %s declared and bound!\n", queue.Name)
	gamelogic.PrintServerHelp()

	// server REPL (read-eval-print loop)
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			log.Println("publishing pause game state")
			publishJSON(pubCh, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
		case "resume":
			log.Println("publishing resume game state")
			publishJSON(pubCh, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
		case "quit":
			log.Println("quitting server")
			return
		default:
			log.Printf("unknown command: %s", words[0])
		}
	}
}

func publishJSON(ch *amqp.Channel, exchange string, key string, val routing.PlayingState) {
	err := pubsub.PublishJSON(ch, exchange, key, val)
	if err != nil {
		log.Printf("could not publish message: %v\n", err)
	}
}
