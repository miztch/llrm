.PHONY: build fmt test

build:
	go build -o llrm .

fmt:
	go fmt ./...

test:
	go test ./...
