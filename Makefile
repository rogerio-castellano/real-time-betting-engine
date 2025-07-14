default: create

c:
create:
	kubectl apply -f ./kubernetes/redis-deployment.yaml
	kubectl apply -f ./kubernetes/redis-service.yaml

	kubectl apply -f ./kubernetes/nats.yaml

	kubectl apply -f ./kubernetes/cockroachdb-service.yaml
	kubectl apply -f ./kubernetes/cockroachdb-statefulset.yaml

	kubectl apply -f ./kubernetes/cockroachdb-init.yaml

	kubectl apply -f ./kubernetes/nats-lb.yaml
	kubectl apply -f ./kubernetes/cockroachdb-lb.yaml

	sleep 15

	@docker run --rm -it natsio/nats-box:latest nats -s nats://host.docker.internal:4222 stream add bets_stream --subjects "bets" --storage memory --retention workq --max-msgs=1000000 --defaults
	@docker run --rm -it natsio/nats-box:latest nats -s nats://host.docker.internal:4222 stream add stats_stream --subjects "stats.update" --storage memory --retention workq --max-msgs=1000000 --defaults

	@kubectl run cockroach-client --rm -i --tty --image=cockroachdb/cockroach:latest-v23.2 --restart=Never --command -- cockroach sql \
	--host=cockroachdb-0.cockroachdb --insecure \
	-e "CREATE TABLE bets(id UUID PRIMARY KEY, game_id STRING NOT NULL, bet_type STRING NOT NULL, amount FLOAT NOT NULL, timestamp TIMESTAMP NOT NULL, pod_id STRING NULL);"

	kubectl apply -f ./kubernetes/backend-deployment.yaml
	kubectl apply -f ./kubernetes/backend-service.yaml

	kubectl apply -f ./kubernetes/stats-aggregator-deployment.yaml
	kubectl apply -f ./kubernetes/stats-aggregator-service.yaml
	kubectl apply -f ./kubernetes/stats-aggregator-lb.yaml

	sleep 10

	# Apply only after your backend is live:
	kubectl apply -f ./kubernetes/hpa.yaml

	sleep 10
