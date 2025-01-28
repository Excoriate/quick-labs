.PHONY: build run stop clean test

# Build both services
build:
	@echo "Building services..."
	docker compose build

# Run the services
run:
	@echo "Starting services..."
	docker compose up

# Run the services in detached mode
run-detached:
	@echo "Starting services in detached mode..."
	docker compose up -d

# Stop the services
stop:
	@echo "Stopping services..."
	docker compose down

# Clean up containers, images, and build cache
clean:
	@echo "Cleaning up..."
	docker compose down --rmi all --volumes --remove-orphans

# Test the services
test:
	@echo "Testing service-a..."
	curl -s -H "X-Auth-Key: service-a-secret-key" http://localhost:8080/greet | jq .message
	@echo "\nTesting service-b..."
	curl -s http://localhost:8081/process | jq .message

# Show logs
logs:
	@echo "Showing logs..."
	docker compose logs -f

# Local Development Targets
.PHONY: run-local-service-a
run-local-service-a:
	@echo "Starting Service A locally..."
	@cd service-a && \
	PORT=8080 \
	AUTH_KEY=service-a-secret-key \
	go run main.go

.PHONY: run-local-service-b
run-local-service-b:
	@echo "Starting Service B locally..."
	@cd service-b && \
	PORT=8081 \
	SERVICE_A_URL=http://localhost:8080 \
	SERVICE_A_AUTH_KEY=service-a-secret-key \
	go run main.go

.PHONY: run-local
run-local: run-local-service-a run-local-service-b

.PHONY: test-local
test-local:
	@echo "Testing local service interaction..."
	@curl -v http://localhost:8081/process | jq .

.PHONY: test-local-greet
test-local-greet:
	@echo "Testing local Service A greeting..."
	@curl -v -H "X-Auth-Key: service-a-secret-key" http://localhost:8080/greet | jq .

# Comprehensive service startup with log inspection and testing
.PHONY: start
start: run-detached wait-for-services test logs

# Wait for services to be fully operational
wait-for-services:
	@echo "Waiting for services to be ready..."
	@for i in {1..30}; do \
		if curl -s http://localhost:8080/health > /dev/null && \
		   curl -s http://localhost:8081/health > /dev/null; then \
			echo "Services are up and running!"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Services did not start in time"; \
	exit 1

.DEFAULT_GOAL := build
