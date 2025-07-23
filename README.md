# 🎯 Real-Time Betting Engine

> 🚧 Work In Progress — This is an evolving real-time engine built for speed, scale, and live insights. Some features may be experimental or incomplete.

A fast, horizontally scalable betting engine designed for real-time ingest, aggregation, and performance visualization — powered by Go, NATS JetStream, PostgreSQL, Redis, and Kubernetes.

> ⚡ Built for speed. 📊 Tuned for stats. 🧠 Optimized for concurrency.

**_This a WIP (Work in progress)_**

## 🔥 Features

- **JetStream + Queue Groups**: Load-balanced message consumption with durable delivery
- **PostgreSQL-backed storage**: Accurate persistence of betting data with automatic failover
- **Redis Odds Engine**: Lightweight odds update logic using atomic Redis counters
- **Live Stats Dashboard**: WebSocket-powered UI with real-time metrics, trends, and failure insights
- **Kubernetes Native**: Full deployment stack with autoscaling via Horizontal Pod Autoscaler (HPA)
- **pprof & Prometheus hooks**: Built-in memory profiling and metrics scraping for performance tuning

## 🛠️ Tech Stack

| Layer         | Stack                                |
| :------------ | :----------------------------------- |
| Language      | Go 1.24                              |
| Messaging     | NATS JetStream                       |
| Database      | PostgreSQL                           |
| Key-Value     | Redis                                |
| UI Dashboard  | HTML + WebSocket + Vanilla JS        |
| Orchestration | Kubernetes (HPA enabled)             |
| Observability | `pprof`, Prometheus-ready `/metrics` |
| Container     | Docker & Kubernetes-ready images     |

## 🚀 How It Works

1. Bets are published to the `bets_stream` using JetStream (e.g., via simulator)
1. Backend pods subscribe to the stream via QueueSubscribe, enabling horizontal scaling
1. Each bet is:
   - Persisted to the database
   - Used to update Redis-based odds
   - Emits stats.update messages to `stats_stream`
1. Stats Aggregator merges per-pod metrics and feeds the WebSocket dashboard
1. The dashboard presents live totals, per-second rates, and health indicators

## 📦 Setup

### 1. Deploy Infrastructure

```bash
make create
```

This creates Redis, NATS (JetStream enabled), Postgres, and initializes streams and tables.

### 2. Launch Backend and Aggregator

```bash
make k8sbuild
```

The backend consumes bets, stores them, and updates stats. The aggregator merges pod stats and serves the WebSocket API.

### 3. Simulate Load

```bash
go run load-simulator/main.go 10 1000
```

→ Sends 1,000 bets at 10ms intervals.

## 📊 Live Dashboard

Visit the Real-Time Dashboard to see:

- ✅ Total bets processed
- 💵 Bet volume
- 🔁 Odds updates
- 🧯 Redis & DB failure metrics
- 🎨 Color-coded accuracy bars
- 🔌 WebSocket status banner

<img src="https://via.placeholder.com/600x300?text=Dashboard+Demo" alt="Dashboard Preview" />

## ⚙️ Scaling & HPA

Pods auto-scale based on memory usage:

```yaml
HorizontalPodAutoscaler:
  minReplicas: 3
  maxReplicas: 5
  target:
    averageMemory: 25Mi
```

Manual scaling also supported:

```bash
kubectl scale deployment betting-engine-backend --replicas=5
```

## 📈 Profiling & Metrics

- Enable memory profiling:

```bash
curl http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/heap
```

Expose metrics via /metrics endpoint:

```bash
curl http://localhost:8081/metrics
```

Can be scraped by Prometheus + visualized in Grafana.

## 🎛️ Reset and Debug Tools

Use `make reset` to purge bets, odds, and stream history.

Streams:

```bash
nats stream info bets_stream
nats consumer info bets_stream bets_consumer
```

Database:

```bash
kubectl run pg-client --rm -it --image=postgres -- psql -U postgres
```

Redis:

```bash
kubectl run redis-inspect --rm -it --image=redis -- redis-cli
```

## 📜 License

MIT — see [LICENSE](LICENSE) for full details.
