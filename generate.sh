#!/bin/bash

set -a
source .env 2>/dev/null
set +a

echo "¿Qué deseas hacer?"
echo "  [1] Generar documentación (CLI)"
echo "  [2] Iniciar servidor web"
read -rp "  Selecciona [1/2] (default: 1): " MODE

case "$MODE" in
  2)
    PORT=${2:-${PORT}}
    echo ""
    echo "Iniciando servidor en http://localhost:${PORT} ..."
    ./${OUTPUT_NAME} serve --port "${PORT}" --api-key "${ANTHROPIC_API_KEY}"
    ;;
  *)
    if [ -n "$1" ]; then
      ./${OUTPUT_NAME} generate --project "$1" --api-key "${ANTHROPIC_API_KEY}"
    else
      ./${OUTPUT_NAME} generate --api-key "${ANTHROPIC_API_KEY}"
    fi
    ;;
esac
