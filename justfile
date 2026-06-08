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
