#!/bin/bash
set -e

echo "=== Construyendo frontend Vue ==="
cd web
npm install
npm run build
cd ..

echo ""
echo "=== Construyendo binario Go ==="
go mod tidy
go mod download && go mod verify
go build -o apidocgen ./cmd/main.go

echo ""
echo "✓ Build completo: ./apidocgen"
echo "  Usa: ./apidocgen serve --port 8080"
echo "  O:   ./apidocgen generate"
