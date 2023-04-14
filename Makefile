.PHONY: build test run clean

build:
	GOARCH=arm64 GOOS=darwin go build -ldflags "-s -w" -o xxqg-darwin-arm64
	GOARCH=amd64 GOOS=darwin go build -ldflags "-s -w" -o xxqg-darwin-amd64
	GOARCH=amd64 GOOS=linux go build -ldflags "-s -w" -o xxqg-linux-amd64
	GOARCH=amd64 GOOS=windows go build -ldflags "-s -w" -o xxqg-windows-amd64.exe

test:
	go test -v -race ./...
	go build -v ./...
	go clean

run:
	go run .

clean:
	go clean
	rm xxqg*
