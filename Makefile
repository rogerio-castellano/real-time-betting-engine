default: create

c:
create:
	kubectl apply -f ./kubernetes/redis-deployment.yaml
	kubectl apply -f ./kubernetes/redis-service.yaml

	kubectl apply -f ./kubernetes/nats.yaml

	kubectl apply -f ./kubernetes/postgres-secret.yaml
	kubectl apply -f ./kubernetes/postgres-deployment.yaml
	sleep 10
	kubectl apply -f ./kubernetes/postgres-service.yaml

	kubectl apply -f ./kubernetes/nats-lb.yaml
# 	kubectl apply -f ./kubernetes/cockroachdb-lb.yaml

	sleep 10

	@docker run --rm -it natsio/nats-box:latest nats -s nats://host.docker.internal:4222 stream add bets_stream --subjects "bets" --storage memory --retention workq --max-msgs=1000000 --defaults
	@docker run --rm -it natsio/nats-box:latest nats -s nats://host.docker.internal:4222 stream add stats_stream --subjects "stats.update" --storage memory --retention workq --max-msgs=1000000 --defaults
	kubectl run pg-client \
	--rm -i --tty \
	--image=postgres:17-alpine \
	--restart=Never \
	--env="PGPASSWORD=example" \
	--command -- psql \
	-h postgres-service \
	-U postgres \
	-d postgres \
	-c "CREATE TABLE bets (id UUID PRIMARY KEY, game_id TEXT NOT NULL, bet_type TEXT NOT NULL, amount DOUBLE PRECISION NOT NULL, timestamp TIMESTAMP NOT NULL, pod_id TEXT);"

	kubectl apply -f ./kubernetes/stats-aggregator-deployment.yaml
	kubectl apply -f ./kubernetes/stats-aggregator-service.yaml
	kubectl apply -f ./kubernetes/stats-aggregator-lb.yaml

	kubectl apply -f ./kubernetes/backend-deployment.yaml
	kubectl apply -f ./kubernetes/backend-service.yaml

	sleep 10

	# Apply only after your backend is live:
	kubectl apply -f ./kubernetes/hpa.yaml

	sleep 10

sbe:
	kubectl scale deployment betting-engine-backend --replicas=$(r)
