include ./Makefile.Common

#RULES
$(TOOLS_DIR):
	mkdir -p $@

.PHONY: check-fmt
ensure-fmt: fmt
	@git diff -s --exit-code *.go || (echo "Build failed: a go file is not formated correctly. Run 'make fmt' and update your PR." && exit 1)

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: govet
govet:
	go vet ./...

.PHONY: compilecheck
compilecheck:
	$(GO_ENV)
	go build -v ./...

.PHONY: gotest
test: gofmt govet ensure-fmt
	CGO_ENABLED=0 go test -v ./... -failfast

.PHONY: gomoddownload
gomoddownload:
	go mod download -x

.PHONY: install-gotools
install-gotools: $(TOOLS_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_DIR) $(GOLINT_VERSION) 

.PHONY: golint
golint:
	$(LINT) run --verbose --allow-parallel-runners --timeout=10m 

.PHONY: gotidy
gotidy:
	go mod tidy -compat=1.21

.PHONY: build
build:
	go build -o bin/guestbook cmd/server/main.go

.PHONY: run
run: gofmt build 
	./bin/main 

.PHONY: gorun
gorun: 
	go run cmd/server/main.go
