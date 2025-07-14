package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

var mu sync.Mutex
var podStatsMap = make(map[string]PodStats)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

var merged PodStats

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket error:", err)
		return
	}
	defer conn.Close()

	for {
		time.Sleep(1 * time.Second)
		if err := conn.WriteJSON(merged); err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
	}
}

func mergeStats() PodStats {
	mu.Lock()
	defer mu.Unlock()

	var merged PodStats
	for _, s := range podStatsMap {
		merged.TotalBets += s.TotalBets
		merged.TotalValue += s.TotalValue
		merged.DbFailures += s.DbFailures
		merged.RedisFailures += s.RedisFailures
		merged.BetsPerSecond += s.BetsPerSecond
		merged.DbFailuresPerSecond += s.DbFailuresPerSecond
		merged.RedisFailuresPerSecond += s.RedisFailuresPerSecond
	}
	return merged
}

func main() {
	// --- Connect to NATS JetStream ---
	var nc *nats.Conn
	var err error
	nc, err = nats.Connect("nats://host.docker.internal:4222")

	if err != nil {
		log.Fatalf("Error connecting to NATS: %v.\nCheck if port forwarding is set up (kubectl port-forward svc/nats-service 4222:4222)", err)
	}
	js, _ := nc.JetStream()

	js.QueueSubscribe("stats.update", "aggregator-group", func(msg *nats.Msg) {
		var podStats PodStats
		if err := json.Unmarshal(msg.Data, &podStats); err != nil {
			log.Println("Failed to unmarshal:", err)
			return
		}
		mu.Lock()
		podStatsMap[podStats.PodID] = podStats
		mu.Unlock()
	})

	var last PodStats

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			merged = mergeStats()
			merged.BetsPerSecond = float64(merged.TotalBets - last.TotalBets)
			merged.DbFailuresPerSecond = float64(merged.DbFailures - last.DbFailures)
			merged.RedisFailuresPerSecond = float64(merged.RedisFailures - last.RedisFailures)
			last = merged
		}
	}()

	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Server started on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
