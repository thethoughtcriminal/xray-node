.PHONY: build test install fmt

build:
	go build -o bin/xray-node ./cmd/xray-node

test:
	go test ./...

fmt:
	gofmt -w .

install: build
	install -m 0755 bin/xray-node /usr/local/bin/xray-node
