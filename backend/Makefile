.PHONY: run build tidy compose-up compose-down

compose-up:
	docker compose up -d

compose-down:
	docker compose down

run:
	go run ./cmd/server

build:
	go build -o bin/sub2video-server ./cmd/server

tidy:
	go mod tidy

linux:
	GOOS=linux GOARCH=amd64 go build -o bin/sub2video-server-linux ./cmd/server
