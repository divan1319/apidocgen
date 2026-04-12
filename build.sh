#!/bin/bash
set -e

usage() {
	cat <<'EOF'
Uso: ./build.sh [opciones]

  (sin opciones)   Build local: requiere Node.js/npm y Go 1.22+.
  --docker, -d     Build con Docker; deja el binario ./apidocgen (solo requiere Docker).
  -h, --help       Muestra esta ayuda.

Ejemplos:
  ./build.sh
  ./build.sh --docker
EOF
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "error: '$1' no está instalado o no está en PATH." >&2
		echo "  Instálalo o ejecuta: ./build.sh --docker" >&2
		exit 1
	fi
}

build_local() {
	require_cmd node
	require_cmd npm
	require_cmd go

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
}

build_docker() {
	require_cmd docker

	echo "=== Construyendo con Docker (frontend + Go) ==="
	DOCKER_BUILDKIT=1 docker build \
		--target artifact \
		-o . \
		-f Dockerfile \
		.

	echo ""
	echo "✓ Build completo (Docker): ./apidocgen"
	echo "  Usa: ./apidocgen serve --port 8080"
	echo "  O:   ./apidocgen generate"
	echo ""
	echo "  Imagen ejecutable: docker build -t apidocgen . && docker run --rm apidocgen help"
}

case "${1:-}" in
-h | --help)
	usage
	exit 0
	;;
--docker | -d)
	build_docker
	;;
"")
	build_local
	;;
*)
	echo "error: opción desconocida: $1" >&2
	usage >&2
	exit 1
	;;
esac
