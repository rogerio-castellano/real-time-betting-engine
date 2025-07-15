default: create

c:
create:
	kubectl apply -f ./kubernetes/redis-deployment.yaml
	kubectl apply -f ./kubernetes/redis-service.yaml
	kubectl wait --for=condition=ready pod -l app=redis --timeout=30s

	kubectl apply -f ./kubernetes/nats.yaml
	kubectl wait --for=condition=ready pod -l app=nats --timeout=30s
	kubectl apply -f ./kubernetes/nats-lb.yaml

	kubectl apply -f ./kubernetes/postgres-secret.yaml
	kubectl apply -f ./kubernetes/postgres-deployment.yaml
	kubectl wait --for=condition=ready pod -l app=postgres --timeout=30s
	kubectl apply -f ./kubernetes/postgres-service.yaml
	kubectl wait --for=condition=ready pod -l app=postgres --timeout=30s

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
	kubectl wait --for=condition=ready pod -l app=stats-aggregator --timeout=30s
	kubectl apply -f ./kubernetes/stats-aggregator-service.yaml
	kubectl apply -f ./kubernetes/stats-aggregator-lb.yaml

	kubectl apply -f ./kubernetes/backend-deployment.yaml
	kubectl wait --for=condition=ready pod -l app=betting-engine --timeout=30s
	kubectl apply -f ./kubernetes/backend-service.yaml

	kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
	kubectl patch deployment metrics-server -n kube-system --type=json -p "$$(cat metrics-patch.json)"
	kubectl rollout restart deployment metrics-server -n kube-system
	kubectl wait --for=condition=ready pod -l app=metrics-server --timeout=30s
	kubectl get apiservices | grep metrics
	kubectl top pods

	# Apply only after your backend is live:
	kubectl wait --for=condition=ready pod -l app=betting-engine --timeout=30s
	kubectl apply -f ./kubernetes/hpa.yaml

sbe:
	kubectl scale deployment betting-engine-backend --replicas=$(r)

hpa:
	kubectl delete HorizontalPodAutoscaler betting-engine-hpa
	pause 10
	kubectl apply -f ./kubernetes/hpa.yaml

rs:
reset:
	kubectl scale deployment stats-aggregator --replicas=0 && kubectl scale deployment betting-engine-backend --replicas=0
	kubectl run pg-client \
	--rm -i --tty \
	--image=postgres:17-alpine \
	--restart=Never \
	--env="PGPASSWORD=example" \
	--command -- psql \
	-h postgres-service \
	-U postgres \
	-d postgres \
	-c "TRUNCATE TABLE bets;"
	kubectl scale deployment stats-aggregator --replicas=1 && kubectl scale deployment betting-engine-backend --replicas=3


