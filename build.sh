#!/bin/bash

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go could not be found. Please install Go and try again."
    exit 1
fi

## Install dependencies
go mod tidy
go mod download && go mod verify

# Build the project
go build -o apidocgen cmd/main.go


echo "Build completed. Run with: ./apidocgen"
