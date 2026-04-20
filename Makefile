.PHONY: build run test coverage clean

build:
	go build -o server.exe ./cmd/server

run: build
	./server.exe

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	rm -f server.exe coverage.out coverage.html
	go clean
