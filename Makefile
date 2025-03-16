up:
	docker-compose up --build -d

vendor:
	go mod tidy && go mod vendor