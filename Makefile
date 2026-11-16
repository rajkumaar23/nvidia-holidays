BIN_DIR := bin

.PHONY: all test fmt vet gen dry-run tidy

all: test

test: fmt vet
	go test ./...

fmt:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed on:"; gofmt -l .; exit 1; }

vet:
	go vet ./...

gen:
	go run ./cmd/gen -out docs/nvidia-holidays.ics -json docs/holidays.json

dry-run:
	go run ./cmd/gen -dry-run

tidy:
	go mod tidy
