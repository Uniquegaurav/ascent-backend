.PHONY: run build tidy db-up db-down up down test

run:        ## run the API (expects Postgres reachable via DATABASE_URL)
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

tidy:
	go mod tidy

db-up:      ## start just Postgres
	docker compose up -d db

db-down:
	docker compose down

up:         ## build + run API and Postgres in Docker
	docker compose up --build

down:
	docker compose down

test:
	go test ./...
