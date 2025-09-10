db-ping:
	psql -h localhost -U demo -d demo -c "SELECT 1;"

test:
	go test ./... -race -cover

run:
	go run ./cmd/

run-prod:
	go run ./producer/

cover:
	gocov test ./... > coverage.json
	gocov-html coverage.json > coverage.html

cover-report: cover
	xdg-open coverage.html || open coverage.html || start coverage.html

dc-up:
	docker-compose up -d

dc-down:
	docker-compose down 

git-all:
	git add .
	git commit -m "13"
	git push origin main

clean:
	rm -f coverage.json coverage.html

.PHONY: test run cover cover-report git-all clean db-ping run-prod dc-up dc-down
