db-ping:
	psql -h localhost -U demo -d demo -c "SELECT 1;"

test:
	go test ./... -race -cover

run:
	go run ./cmd/server

cover:
	gocov test ./... > coverage.json
	gocov-html coverage.json > coverage.html

cover-report: cover
	xdg-open coverage.html || open coverage.html || start coverage.html

git-all:
	git add .
	git commit -m "7"
	git push origin main

clean:
	rm -f coverage.json coverage.html
