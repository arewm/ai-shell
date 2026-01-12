.PHONY: build clean test lint

BINARY_NAME=ai-shell

build:
	go build -o $(BINARY_NAME) cmd/ai-shell/*.go

clean:
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

lint:
	golangci-lint run
