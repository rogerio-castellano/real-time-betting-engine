default : k8sbuild

build:
	go build .

k:k8sbuild
k8sbuild:
	docker build -t stats-aggregator .
	kubectl rollout restart deployment stats-aggregator
	@date +"%T"
