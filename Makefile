db-ping:
	psql -h localhost -U demo -d demo -c "SELECT 1;"
test:
	go test ./... -race -cover
run:
	go run ./cmd/server