BINARY := name-frequency-counter
GO     := go

.PHONY: build test lint clean run

## build: compile the binary
build:
	$(GO) build -o $(BINARY) .

## test: run all unit and integration tests with race detector
test:
	$(GO) test -race ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY)

## run: run the utility with FILE variable (usage: make run FILE=names.txt)
run: build
	./$(BINARY) count $(FILE)
