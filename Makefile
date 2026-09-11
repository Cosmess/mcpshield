.PHONY: test test-race vet build dev-up dev-down

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build ./cmd/...

dev-up:
	docker compose up --build -d

dev-down:
	docker compose down