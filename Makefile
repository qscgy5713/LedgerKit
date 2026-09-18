.PHONY: build test lint run clean

BINARY := ledgerkit

build:
	go build -o $(BINARY) ./cmd/ledgerkit

test:
	go test ./... -race

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
