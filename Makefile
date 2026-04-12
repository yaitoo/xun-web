.PHONY: install dev build clean run migrate

# Install Node dependencies
install:
	npm install

# Development mode: Tailwind watch + app
dev:
	npm run dev & go run cmd/app/main.go

# Production build: compile Tailwind CSS
build:
	npm run build

# Watch Tailwind CSS only
watch:
	npm run watch

# Run the application
run:
	go run cmd/app/main.go

# Run database migrations
migrate:
	go run cmd/app/main.go -migrate

# Clean compiled assets
clean:
	npm run clean

# Format Go code
fmt:
	go fmt ./...

# Tidy Go modules
tidy:
	go mod tidy

# Build production binary
build-binary:
	go build -o bin/xun-web ./cmd/app
