default:
    just --list

run:
    go run ./cmd/lobsters-planet

fmt:
    gofumpt -w .

lint:
    golangci-lint run

test:
    go test ./...

discover:
    go run ./cmd/lobsters-planet discover
