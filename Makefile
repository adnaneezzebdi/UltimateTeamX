PROTO_DIR := proto
DOCKER_COMPOSE := docker/docker-compose.yaml

.PHONY: proto up down test tidy run-market run-club-mock

proto:
	buf generate

up:
	docker compose -f $(DOCKER_COMPOSE) up -d

down:
	docker compose -f $(DOCKER_COMPOSE) down

test:
	go test ./...

tidy:
	go mod tidy

run-market:
	GO_DOTENV_PATH=.env go run service/market/cmd/server/main.go

