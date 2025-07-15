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
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
)

// Represents a bet coming into the system
type Bet struct {
	ID        string    `json:"id"`
	GameID    string    `json:"game_id"`
	BetType   string    `json:"bet_type"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

var stats = PodStats{}
var ctx = context.Background()
var podID = uuid.New().String()

func main() {
	// --- Connect to NATS JetStream ---
	var nc *nats.Conn
	var err error
	log.Println("Connecting to NATS...")
	nc, err = nats.Connect(os.Getenv("NATS_URL"))

	if err != nil {
		log.Fatalf("Error connecting to NATS: %v.\nCheck if port forwarding is set up (kubectl port-forward svc/nats-service 4222:4222)", err)
	}
	defer nc.Close()
	js, _ := nc.JetStream()

	// --- Connect to Postgres ---
	var db *sql.DB
	dbURL := os.Getenv("POSTGRES_URL")
	log.Println(dbURL)
	log.Println("Connecting to Postgres...")
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("sql.Open failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Println("Calling db.PingContext...")
	for i := range 5 {
		start := time.Now()
		err = db.PingContext(ctx)
		log.Printf("Ping duration: %v", time.Since(start))
		if err == nil {
			break
		}
		log.Printf("Ping attempt %d failed: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Ping failed after retries: %v", err)
	}

	// --- Connect to Redis ---
	log.Println("Connecting to Redis...")
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

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

		// stats.PodID = os.Getenv("POD_NAME") // Or use hostname
		stats.PodID = podID
		statsJSON, _ := json.Marshal(stats)
		nc.Publish("stats.update", statsJSON)
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM bets;").Scan(&count)
		if err != nil {
			log.Printf("DB query failed: %v", err) // Add this!
			http.Error(w, "DB query failed", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]int{"total_bets": count})
	})

	log.Println("Server started on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func storeBet(db *sql.DB, bet Bet) {
	sqlStatement := `INSERT INTO bets (id, game_id, bet_type, amount, timestamp, pod_id) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(sqlStatement, bet.ID, bet.GameID, bet.BetType, bet.Amount, bet.Timestamp, podID)
	if err != nil {
		log.Printf("Error storing bet: (id:%v) - %v", bet.ID, err)
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
