package main

import (
	"fmt"
	"log"
	"strconv"

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
	log.Println("Peril game client connected to RabbitMQ!")

	pubCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}
	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.QTypeTransient,
		handlerMove(gs, pubCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.QTypeDurable,
		handlerWar(gs, pubCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war declarations: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.QTypeTransient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	// client REPL (read-eval-print loop)
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "move":
			mv, err := gs.CommandMove(words)
			if err != nil {
				log.Println(err)
				continue
			}
			err = pubsub.PublishJSON(
				pubCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+mv.Player.Username,
				mv,
			)
			if err != nil {
				log.Printf("error publish message: %v\n", err)
				continue
			}
			log.Printf("moved %d units to %s\n", len(mv.Units), mv.ToLocation)
		case "spawn":
			err = gs.CommandSpawn(words)
			if err != nil {
				log.Println(err)
				continue
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(words) < 2 {
				fmt.Println("usage: spam <n>")
				continue
			}
			n, err := strconv.Atoi(words[1])
			if err != nil {
				fmt.Printf("error: %s is not a valid number\n", words[1])
				continue
			}
			for range n {
				msg := gamelogic.GetMaliciousLog()
				err = publishGameLog(pubCh, username, msg)
				if err != nil {
					fmt.Printf("error publishing malicious log: %s\n", err)
				}
			}
			fmt.Printf("Published %v malicious logs\n", n)
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			log.Printf("unknown command: %s", words[0])
		}
	}
}
