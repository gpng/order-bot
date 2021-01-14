# Docker parameters
DOCKERCMD=docker
DOCKERCOMPOSECMD=docker-compose

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

# App parameters
CONTAINER_TAG=order-bot
DOCKERFILE=Dockerfile.dev
MAIN_FOLDER=cmd/api
MAIN_PATH=$(MAIN_FOLDER)/main.go

# Goose parameters for migrations
GOOSECMD=goose
DBSTRING="host=${DB_HOST} user=${DB_USER} dbname=${DB_NAME} sslmode=disable password=${DB_PASSWORD}"
APISCHEMAPATH=sqlc/schemas/

# sqlc parameters
SQLCCMD=sqlc

default: build up logs

build:
	@echo "=============building API============="
	$(DOCKERCMD) build -f $(DOCKERFILE) -t $(CONTAINER_TAG) .

up:
	@echo "=============starting API locally============="
	$(DOCKERCOMPOSECMD) up -d

logs:
	$(DOCKERCOMPOSECMD) logs -f

run:
	go build -o bin/application $(MAIN_PATH) && ./bin/application -docs

down:
	$(DOCKERCOMPOSECMD) down --remove-orphans

test:
	godotenv -f .test.env $(GOTEST) -cover ./... -count=1

clean: down
	@echo "=============cleaning up============="
	$(DOCKERCMD) system prune -f
	$(DOCKERCMD) volume prune -f

run-prod:
	$(DOCKERCMD) build -t order-bot-eb .
	docker run -p 4000:5000 order-bot-eb

gen-docs:
	swag init -g $(MAIN_PATH)

gen-models:
	$(SQLCCMD) generate

migrate:
	$(GOOSECMD) -dir $(APISCHEMAPATH) postgres $(DBSTRING) up

rollback:
	$(GOOSECMD) -dir $(APISCHEMAPATH) postgres $(DBSTRING) down