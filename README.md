# Demo Service
Микросервис на Go для обработки заказов из Kafka, хранения в PostgreSQL и API.

## Запуск
1. `docker-compose up -d` — запустить PostgreSQL.
2. `go run ./cmd/server` — запустить сервис.

## Требования
- Go 1.21
- Docker, docker-compose