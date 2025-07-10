package main

import (
	"encoding/json"
	"log"
	"math/rand"
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
		if err := nc.Publish("bets", betJSON); err != nil {
			log.Printf("Error publishing bet: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Adjust for desired throughput
	}
}
