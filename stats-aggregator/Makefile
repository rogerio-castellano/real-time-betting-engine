default : k8sbuild

build:
	go build .

k:
k8sbuild:
	docker build -t stats-aggregator .
	kubectl scale deployment stats-aggregator --replicas=0
	@sleep 1
	kubectl scale deployment stats-aggregator --replicas=1
	@echo "Press Enter to continue..."
	@read
	kubectl get pods -l app=stats-aggregator