run:
	DB_PATH=data/app.db APP_SECRET=local go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/app ./cmd/server

fmt:
	gofmt -s -w .

tidy:
	go mod tidy
