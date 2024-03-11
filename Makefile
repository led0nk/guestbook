build:
	go build -o bin/main cmd/server/main.go

run: fmt build 
	./bin/main

fmt:
	go fmt ./...

test: fmt
	go test -v ./... -failfast
