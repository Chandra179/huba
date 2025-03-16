.PHONY: run keycloak clean

# Run the example application
run:
	go run main.go

# Start Keycloak using Docker Compose
keycloak:
	docker-compose up -d

# Stop Keycloak and remove containers
clean:
	docker-compose down

# Start both Keycloak and the application
all: keycloak run 