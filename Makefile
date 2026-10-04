BINARY := drobo

.PHONY: test build clean

test:
	go test ./...
	go vet ./...

build:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/drobo

clean:
	rm -f $(BINARY)
