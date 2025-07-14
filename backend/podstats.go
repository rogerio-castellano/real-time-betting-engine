package main

type PodStats struct {
	PodID                  string  `json:"pod_id"`
	TotalBets              int     `json:"total_bets"`
	TotalValue             float64 `json:"total_value"`
	DbFailures             int     `json:"db_failures"`
	RedisFailures          int     `json:"redis_failures"`
	BetsPerSecond          float64 `json:"bets_per_second"`
	DbFailuresPerSecond    float64 `json:"db_failures_per_second"`
	RedisFailuresPerSecond float64 `json:"redis_failures_per_second"`
}
