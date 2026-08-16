.PHONY: test build run-api run-worker
test:
	go test ./...
build:
	go build ./cmd/api ./cmd/worker
run-api:
	go run ./cmd/api
run-worker:
	go run ./cmd/worker
