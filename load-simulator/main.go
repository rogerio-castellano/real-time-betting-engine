package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type Bet struct {
	ID        string    `json:"id"`
	GameID    string    `json:"game_id"`
	BetType   string    `json:"bet_type"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	nc, err := nats.Connect("nats://localhost:4222") // Connect to NATS via port-forward
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		log.Fatalf("Error getting JetStream Context: %v", err)
	}

	ms, _ := strconv.ParseInt(os.Args[1], 10, 64)
	if ms == 0 {
		ms = 10
	}

	bets, _ := strconv.Atoi(os.Args[2])
	if bets == 0 {
		ms = 10000
	}
	counter := 0

	log.Println("Load simulator started...")
	for {
		bet := Bet{
			ID:        uuid.New().String(),
			GameID:    "game_123",
			BetType:   "moneyline",
			Amount:    float64(rand.Intn(100) + 1),
			Timestamp: time.Now(),
		}
		betJSON, _ := json.Marshal(bet)
		js.PublishAsync("bets", betJSON)
		time.Sleep(time.Duration(ms) * time.Millisecond) // Adjust for desired throughput

		counter++
		if counter >= bets {
			<-js.PublishAsyncComplete()
			return
		}
	}

}
