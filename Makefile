.PHONY: all test lint

all: build

build:
	@echo "Building..."
	go build ./...

test:
	go test -race ./...
	cd grpcerr && go test -race ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint is not installed"; \
		exit 1; \
	}
	golangci-lint run ./...
	cd grpcerr && golangci-lint run ./...
