BINARY_NAME=mandatory-wait-plugin
DOCKER_IMAGE=argo-rollouts-mandatory-wait-plugin
DOCKER_TAG=latest

.PHONY: build clean test docker-build docker-push build-mandatory-wait build-mandatory-wait-local

# Build the mandatory wait plugin
build:
	CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -a -installsuffix cgo -o $(BINARY_NAME) ./step-plugin-mandatory-wait

# Build mandatory wait plugin for local development
build-local:
	go build -buildvcs=false -o $(BINARY_NAME) ./step-plugin-mandatory-wait

# Build the mandatory wait plugin (alias)
build-mandatory-wait: build

# Build mandatory wait plugin for local development (alias)
build-mandatory-wait-local: build-local

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Run tests
test:
	go test ./...

# Run go mod tidy
tidy:
	go mod tidy

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Push Docker image (requires docker login)
docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Run locally for testing
run-local: build-local
	./$(BINARY_NAME)

# Install dependencies
deps:
	go mod download

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Build everything
all: clean deps build test