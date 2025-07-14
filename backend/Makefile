build:
b:
	go build .

k8sbuild:
k:
	docker build -t real-time-betting-engine .
	kubectl scale deployment betting-engine-backend --replicas=0
	kubectl scale deployment betting-engine-backend --replicas=3
	@echo "Press Enter to continue..."
	@read
	kubectl get pods -l app=betting-engine
