package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// Represents a bet coming into the system
type Bet struct {
	ID        string    `json:"id"`
	GameID    string    `json:"game_id"`
	BetType   string    `json:"bet_type"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

// Live statistics to be sent to the dashboard
type LiveStats struct {
	TotalBets     int     `json:"total_bets"`
	BetsPerSecond float64 `json:"bets_per_second"`
	TotalValue    float64 `json:"total_value"`
}

var stats = LiveStats{}
var ctx = context.Background()

func main() {
	// --- Connect to NATS JetStream ---
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	js, _ := nc.JetStream()

	// --- Connect to CockroachDB ---
	db, err := sql.Open("postgres", "postgresql://root@localhost:26257/defaultdb?sslmode=disable")
	if err != nil {
		log.Fatalf("Error connecting to CockroachDB: %v", err)
	}
	defer db.Close()

	// --- Connect to Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Subscribe to the "bets" stream
	js.Subscribe("bets", func(msg *nats.Msg) {
		msg.Ack()
		var bet Bet
		if err := json.Unmarshal(msg.Data, &bet); err != nil {
			log.Printf("Error unmarshaling bet: %v", err)
			return
		}

		// 1. Process and store the bet in CockroachDB
		go storeBet(db, bet)

		// 2. Update odds in Redis (example logic)
		go updateOdds(rdb, bet.GameID)

		// 3. Update live stats
		stats.TotalBets++
		stats.TotalValue += bet.Amount
	})

	// --- WebSocket Handler for Live Stats ---
	http.HandleFunc("/ws", handleWebSocket)

	// Ticker to calculate bets per second
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		var lastBetCount int
		for range ticker.C {
			stats.BetsPerSecond = float64(stats.TotalBets - lastBetCount)
			lastBetCount = stats.TotalBets
		}
	}()

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func storeBet(db *sql.DB, bet Bet) {
	sqlStatement := `INSERT INTO bets (id, game_id, bet_type, amount, timestamp) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(sqlStatement, bet.ID, bet.GameID, bet.BetType, bet.Amount, bet.Timestamp)
	if err != nil {
		log.Printf("Error storing bet: %v", err)
	}
}

func updateOdds(rdb *redis.Client, gameID string) {
	// In a real system, you'd have complex logic here.
	// For this showcase, we'll just increment a key.
	err := rdb.Incr(ctx, fmt.Sprintf("game:%s:odds_updates", gameID)).Err()
	if err != nil {
		log.Printf("Error updating odds in Redis: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// Push stats to the dashboard every second
	for {
		time.Sleep(1 * time.Second)
		if err := conn.WriteJSON(stats); err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
	}
}
