package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/go-redis/redis/v8"
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

var stats = LoadSnapshot{}

var ctx = context.Background()
var podID = os.Getenv("HOSTNAME")
var mu sync.Mutex

func storeBet(db *sql.DB, bet Bet) {
	sqlStatement := `INSERT INTO bets (id, game_id, bet_type, amount, timestamp, pod_id) VALUES ($1, $2, $3, $4, $5, $6)`

	err := retryWithBackoff(func() error {
		_, err := db.Exec(sqlStatement, bet.ID, bet.GameID, bet.BetType, bet.Amount, bet.Timestamp, podID)
		return err
	}, 5, 1*time.Second)

	if err != nil {
		log.Printf("Error storing bet: (id:%v) - %v", bet.ID, err)
		mu.Lock()
		stats.DbFailures++
		mu.Unlock()
	}
}

func updateOdds(rdb *redis.Client, gameID string) {
	// In a real system, you'd have complex logic here.
	// For this showcase, we'll just increment a key.

	err := rdb.Incr(ctx, fmt.Sprintf("game:%s:odds_updates", gameID)).Err()
	if err != nil {
		mu.Lock()
		stats.RedisFailures++
		mu.Unlock()
	}
}

func retryWithBackoff(task func() error, maxRetries int, baseDelay time.Duration) error {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := task()
		if err == nil {
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("💥 all retries failed: %w", err)
		}

		jitter := time.Duration(rand.Intn(100)) * time.Millisecond
		backoff := time.Duration(1<<attempt) * baseDelay
		wait := backoff + jitter

		fmt.Printf("🔁 Retry %d in %v...\n", attempt+1, wait)
		time.Sleep(wait)
	}
	return nil
}

func dbConnection(ctx context.Context) (*sql.DB, error) {
	var db *sql.DB

	dbURL := os.Getenv("POSTGRES_URL")
	log.Println("Connecting to Postgres...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("sql.Open failed: %v", err)
	}

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
		return nil, fmt.Errorf("ping failed after retries: %v", err)
	}

	return db, nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var count int
	db, err := dbConnection(r.Context())
	if err != nil {
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM bets;").Scan(&count)
	if err != nil {
		log.Printf("DB query failed: %v", err)
		http.Error(w, "DB query failed", http.StatusInternalServerError)
		return
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		log.Printf("Error getting JetStream Context: %v", err)
	}

	stream, err := js.StreamInfo("bets_stream")
	if err != nil {
		http.Error(w, "Get Bets stream failed", http.StatusInternalServerError)
		log.Printf("Get Bets stream failed: %v", err)
	}

	pendingBets := int(stream.State.Msgs)

	gameID := "game_123"
	redisCount, err := rdb.Get(ctx, fmt.Sprintf("game:%s:odds_updates", gameID)).Int()
	if err != nil && err != redis.Nil {
		http.Error(w, "Get Odds update failed", http.StatusInternalServerError)
		log.Printf("Get Odds update failed: %v", err)
	}

	json.NewEncoder(w).Encode(map[string]int{"bets_table_row_count": count,
		"total_odds":   redisCount,
		"pending_bets": pendingBets})
}

var nc *nats.Conn
var rdb *redis.Client

func main() {
	ncLocal, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v.\nCheck if port forwarding is set up (kubectl port-forward svc/nats-service 4222:4222)", err)
	}
	nc = ncLocal

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		log.Fatalf("Error getting JetStream Context: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := dbConnection(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(30)

	log.Println("Connecting to Redis...")
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		PoolSize: 20, // max connections
	})
	defer rdb.Close()

	// Subscribe to the "bets" stream
	// js.QueueSubscribe("bets", "betting-engine-group", func(msg *nats.Msg) {
	js.Subscribe("bets", func(msg *nats.Msg) {
		log.Println("Processing queue!")
		msg.Ack()
		var bet Bet
		if err := json.Unmarshal(msg.Data, &bet); err != nil {
			log.Printf("Error unmarshaling bet: %v", err)
			return
		}
		if time.Now().Nanosecond()%100000 == 0 {
			log.Print(&bet)
		}
		// 1. Process and store the bet in the database
		go storeBet(db, bet)

		// 2. Update odds in Redis (example logic)
		go updateOdds(rdb, bet.GameID)

		// 3. Update live stats
		stats.TotalBets++
		stats.TotalValue += bet.Amount
		stats.PodID = podID

		statsJSON, _ := json.Marshal(stats)
		nc.Publish("stats.update", statsJSON)
	}, nats.Durable("bets_consumer"), nats.BindStream("bets_stream"))

	http.HandleFunc("/stats", statsHandler)

	if os.Getenv("PROFILER") != "" {
		log.Println("Running Profiler")
		go http.ListenAndServe(":6060", nil)
	}
	log.Println("Server started on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
