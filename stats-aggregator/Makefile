build:
	go build .

k8sbuild:
k:
	docker build -t stats-aggregator .
	kubectl scale deployment stats-aggregator --replicas=0
	kubectl scale deployment stats-aggregator --replicas=1
	@echo "Press Enter to continue..."
	@read
	kubectl get pods -l app=stats-aggregator