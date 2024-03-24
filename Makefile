.PHONY: build
build:
	go build -o bin/main cmd/server/main.go

.PHONY: run
run: fmt build 
	./bin/main 

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test: fmt
	go test -v ./... -failfast

.PHONY: lint
lint:
	golangci-lint run --verbose --timeout=10m ./...

.PHONY: tidy
	tidy:
	go mod tidy -v -compat=1.22
