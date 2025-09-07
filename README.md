# Demo Service
Микросервис на Go для обработки заказов из Kafka, хранения в PostgreSQL и предоставления API.

## Запуск
1. `make dc-up` - запустить все сервисы (PostgreSQL, Zookeeper, Kafka).
2. `make run` - запустить сервис.
3. `make run-prod` - запустить скрипт эмулятор

## Остановка
- `make dc-down` - остановить и удалить контейнеры.

## Требования
- Go 1.21
- Docker, docker-compose

## Использование
- API: `GET http://localhost:8081/order/<order_uid>` - получить заказ.
- Интерфейс: `http://localhost:8081` для ввода ID заказа.
