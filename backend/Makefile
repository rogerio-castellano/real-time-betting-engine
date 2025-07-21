default : k8sbuild

b:build
build:
	go build .

k:k8sbuild
k8sbuild:
	docker build -t real-time-betting-engine .
	kubectl rollout restart deployment betting-engine-backend
	kubectl rollout restart deployment stats-aggregator
	@date +"%T"

