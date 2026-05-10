BINARY := name-frequency-counter
GO     := go

.PHONY: build test lint clean run

build:
	$(GO) build -o $(BINARY) .

test:
	$(GO) test -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

run: build ./$(BINARY) count $(FILE)
