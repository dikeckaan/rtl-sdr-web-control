.PHONY: build dev clean test

BINARY=sdr-server
GOFLAGS=-trimpath -ldflags="-s -w"

build:
	cd web/frontend && npm ci && npm run build
	go build $(GOFLAGS) -o $(BINARY) ./cmd/sdr-server

build-backend:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/sdr-server

dev:
	go run ./cmd/sdr-server -config config.toml

test:
	go test ./...

clean:
	rm -f $(BINARY)
	rm -rf web/frontend/dist

install-deps:
	sudo apt-get update && sudo apt-get install -y \
		rtl-sdr ffmpeg multimon-ng sox

deploy: build
	sudo cp $(BINARY) /usr/local/bin/
	sudo cp config.toml /etc/sdr/config.toml
	sudo systemctl restart sdr
