package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
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
	TotalBets              int     `json:"total_bets"`
	BetsPerSecond          float64 `json:"bets_per_second"`
	TotalValue             float64 `json:"total_value"`
	DbFailures             int     `json:"db_failures"`
	RedisFailures          int     `json:"redis_failures"`
	DbFailuresPerSecond    float64 `json:"db_failures_per_second"`
	RedisFailuresPerSecond float64 `json:"redis_failures_per_second"`
}

var stats = LiveStats{}
var ctx = context.Background()

func main() {
	runInContainer := os.Getenv("REDIS_URL") != ""
	// --- Connect to NATS JetStream ---
	var nc *nats.Conn
	var err error
	if runInContainer {
		nc, err = nats.Connect("nats://host.docker.internal:4222")
	} else {
		nc, err = nats.Connect("nats://localhost:4222")
	}

	if err != nil {
		log.Fatalf("Error connecting to NATS: %v.\nCheck if port forwarding is set up (kubectl port-forward svc/nats-service 4222:4222)", err)
	}
	defer nc.Close()
	js, _ := nc.JetStream()

	// --- Connect to CockroachDB ---
	var db *sql.DB
	if runInContainer {
		dbURL := os.Getenv("COCKROACHDB_URL")
		db, err = sql.Open("postgres", dbURL)
	} else {
		db, err = sql.Open("postgres", "postgresql://root@localhost:26257/defaultdb?sslmode=disable")
	}

	if err != nil {
		log.Fatalf("Error connecting to CockroachDB: %v", err)
	}
	defer db.Close()

	// --- Connect to Redis ---
	var rdb *redis.Client
	if runInContainer {
		rdb = redis.NewClient(&redis.Options{
			Addr: os.Getenv("REDIS_URL"),
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
	}
	// Subscribe to the "bets" stream
	js.QueueSubscribe("bets", "betting-engine-group", func(msg *nats.Msg) {
		msg.Ack()
		var bet Bet
		if err := json.Unmarshal(msg.Data, &bet); err != nil {
			log.Printf("Error unmarshaling bet: %v", err)
			return
		}
		if time.Now().Nanosecond()%100000 == 0 {
			log.Print(&bet)
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
		var lastDBFailuresCount int
		var lastRedisFailuresCount int
		for range ticker.C {
			stats.BetsPerSecond = float64(stats.TotalBets - lastBetCount)
			lastBetCount = stats.TotalBets
			stats.DbFailuresPerSecond = float64(stats.DbFailures - lastDBFailuresCount)
			lastDBFailuresCount = stats.DbFailures
			stats.RedisFailuresPerSecond = float64(stats.RedisFailures - lastRedisFailuresCount)
			lastRedisFailuresCount = stats.RedisFailures
		}
	}()

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func storeBet(db *sql.DB, bet Bet) {
	sqlStatement := `INSERT INTO bets (id, game_id, bet_type, amount, timestamp) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(sqlStatement, bet.ID, bet.GameID, bet.BetType, bet.Amount, bet.Timestamp)
	if err != nil {
		// log.Printf("Error storing bet: %v", err)
		stats.DbFailures++
	}
}

func updateOdds(rdb *redis.Client, gameID string) {
	// In a real system, you'd have complex logic here.
	// For this showcase, we'll just increment a key.
	err := rdb.Incr(ctx, fmt.Sprintf("game:%s:odds_updates", gameID)).Err()
	if err != nil {
		// log.Printf("Error updating odds in Redis: %v", err)
		stats.RedisFailures++
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
