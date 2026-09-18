.PHONY: build build-server test lint run run-server clean

BINARY := ledgerkit
SERVER_BINARY := ledgerkit-server

build:
	go build -o $(BINARY) ./cmd/ledgerkit

build-server:
	go build -o $(SERVER_BINARY) ./cmd/ledgerkit-server

test:
	go test ./... -race

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...

run: build
	./$(BINARY)

run-server: build-server
	./$(SERVER_BINARY)

clean:
	rm -f $(BINARY) $(SERVER_BINARY)
