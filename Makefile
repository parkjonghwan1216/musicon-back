.PHONY: build run test vet fmt

build:
	go build -o musicon-back ./cmd/server

run:
	go run ./cmd/server

test:
	go test -race -cover ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .
