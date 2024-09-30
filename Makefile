.PHONY: build test format

build:
	go build -o build

test:
	go test -v

format:
	gofmt -w -s .