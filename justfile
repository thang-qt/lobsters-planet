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

feeds:
    go run ./cmd/lobsters-planet feeds

build:
    go run ./cmd/lobsters-planet build

serve:
    nix shell nixpkgs#python3 -c python3 -m http.server 8080 --directory public
