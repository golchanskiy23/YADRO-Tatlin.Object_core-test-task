.PHONY: build test lint clean run

build:
	go build -o name-frequency-counter ./cmd/

test-race:
	go test -race ./...

test-cover:
	go test -cover ./...

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f name-frequency-counter

# Usage:
#   make run FILE=names.txt
#   make run FILE=names.txt TOP=10
run: 
	go run ./cmd/main.go count $(if $(TOP),--top $(TOP),) $(FILE)
