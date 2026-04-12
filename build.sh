#!/bin/bash

set -a
source .env 2>/dev/null
set +a
# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go could not be found. Please install Go and try again."
    exit 1
fi

## Install dependencies
go mod tidy
go mod download && go mod verify

# Build the project
go build -o ${OUTPUT_NAME} cmd/main.go


echo "Build completed. Run with: ./${OUTPUT_NAME}"
