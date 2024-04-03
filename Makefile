# Makefile

# Application Name
APP_NAME=stay-alive

# Build binary for host OS
build:
	go build -o $(APP_NAME)

# Build binary for macOS with M2 chip
mac-m2:
	GOOS=darwin GOARCH=arm64 go build -o $(APP_NAME)-m2

.PHONY: build mac-m2
