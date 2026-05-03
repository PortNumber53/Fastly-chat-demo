.PHONY: build run test clean docker

# Build the binary
build:
	go build -o chat-demo .

# Run in development mode
run: build
	./chat-demo

# Run with Fanout enabled (requires env vars)
run-fanout: build
	FASTLY_FANOUT_ENABLED=true ./chat-demo

# Run tests
test:
	go test ./...

# Format and lint
lint:
	go fmt ./...
	go vet ./...

# Clean build artifacts
clean:
	rm -f chat-demo

# Docker build and run
docker:
	docker compose up --build

# Docker build only
docker-build:
	docker build -t fastly-chat-demo .
