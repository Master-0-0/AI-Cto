.PHONY: build-teamserver build-implant-windows build-implant-linux build-web dev clean

# Build teamserver binary
build-teamserver:
	go build -o bin/teamserver ./cmd/teamserver

# Build Windows implant (amd64)
build-implant-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o payloads/implant-windows-amd64.exe ./cmd/implant

# Build Linux implant (amd64)
build-implant-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o payloads/implant-linux-amd64 ./cmd/implant

# Build web frontend
build-web:
	cd web && npm install && npm run build

# Run teamserver in development mode
dev:
	go run ./cmd/teamserver --config=config/config.yaml

# Clean build artifacts
clean:
	rm -rf bin/ payloads/ web/dist/
