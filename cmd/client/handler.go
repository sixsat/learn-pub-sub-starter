package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerMove(gs *gamelogic.GameState, pubCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")

		moveOutcome := gs.HandleMove(move)
		switch moveOutcome {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.Ack
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				pubCh,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				log.Printf("error publish message: %v\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			log.Println("error: unknown move outcome")
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState, pubCh *amqp.Channel) func(dw gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(w gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(w)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			err := publishGameLog(
				pubCh,
				gs.GetUsername(),
				fmt.Sprintf("%s won a war against %s", winner, loser),
			)
			if err != nil {
				log.Printf("error publish message: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			err := publishGameLog(
				pubCh,
				gs.GetUsername(),
				fmt.Sprintf("%s won a war against %s", winner, loser),
			)
			if err != nil {
				log.Printf("error publish message: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			err := publishGameLog(
				pubCh,
				gs.GetUsername(),
				fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser),
			)
			if err != nil {
				log.Printf("error publish message: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			log.Println("error: unknown war outcome")
			return pubsub.NackDiscard
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func publishGameLog(pubCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		pubCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}
