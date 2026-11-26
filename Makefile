.PHONY: build release vet test
build:
	CGO_ENABLED=0 go build -o codgo ./cmd/codgo

# Small static binary for the target machine.
release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/codgo-linux-amd64 ./cmd/codgo

vet:
	go vet ./...

test:
	go test ./...
