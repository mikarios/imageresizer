.PHONY: test lint lint118 prepare-image-lambda build build-imagemaker-linux check-up-to-date

test:
	go test ./...

lint:
	golangci-lint run -c .golangci.yml

lint118:
	golangci-lint run -c .golangci.yml --disable gocritic

prepare-image-lambda:
	mkdir -p bin
	go build -o bin/imagelambda ./cmd/processimagelambda
	zip bin/imagelambda.zip bin/imagelambda

check-up-to-date:
	git remote update && git status -uno | grep -E "Your branch is up to date|Your branch is ahead of" || (echo "\033[0;31myou should git pull\033[0m"; exit 1)

build: check-up-to-date
	mkdir -p bin
	go build -o bin/imageResizer ./cmd/imageResizer

build-linux: check-up-to-date
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/imageResizer ./cmd/imageResizer