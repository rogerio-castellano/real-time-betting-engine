default : k8sbuild

build:
b:
	go build .

k:
k8sbuild:
	docker build -t real-time-betting-engine .
	kubectl scale deployment betting-engine-backend --replicas=0
	kubectl scale deployment betting-engine-backend --replicas=3
	@echo "Press Enter to continue..."
	@read
	kubectl get pods -l app=betting-engine
