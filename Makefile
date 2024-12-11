.PHONY: build test format lint

build:
	go build -o build

test:
	go test -v ./...

format:
	gofmt -w -s .

lint:
	golangci-lint run --fix