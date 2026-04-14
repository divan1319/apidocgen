#!/bin/bash

set -a
source .env 2>/dev/null
set +a

# Si .env no define OUTPUT_NAME, usar el nombre por defecto del binario
OUTPUT_NAME="${OUTPUT_NAME:-apidocgen}"

if [ ! -e "./${OUTPUT_NAME}" ]; then
  echo "error: no se encuentra ./${OUTPUT_NAME}" >&2
  echo "  Compila con build.sh o define OUTPUT_NAME en .env (nombre del binario generado)." >&2
  exit 1
fi

echo "¿Qué deseas hacer?"
echo "  [1] Generar documentación (CLI)"
echo "  [2] Iniciar servidor web"
read -rp "  Selecciona [1/2] (default: 1): " MODE

case "$MODE" in
  2)
    PORT=${2:-${PORT}}
    echo ""
    echo "Iniciando servidor en http://localhost:${PORT} ..."
    echo "  (Las claves se leen del entorno: ANTHROPIC_API_KEY, OPENAI_API_KEY, DEEPSEEK_API_KEY.)"
    echo "  Cada proyecto usa la clave del proveedor que tenga configurado en el panel."
    ./${OUTPUT_NAME} serve --port "${PORT}"
    ;;
  *)
    echo ""
    echo "Proveedor de IA para esta generación:"
    echo "  [1] Anthropic — variable ANTHROPIC_API_KEY en .env"
    echo "  [2] OpenAI — variable OPENAI_API_KEY en .env"
    echo "  [3] DeepSeek — variable DEEPSEEK_API_KEY en .env"
    read -rp "  Selecciona [1-3] (default: 1): " PROV_CHOICE
    PROV_CHOICE="${PROV_CHOICE:-1}"
    case "$PROV_CHOICE" in
      2|openai|OpenAI)
        AI_PROVIDER="openai"
        API_KEY="${OPENAI_API_KEY}"
        KEY_NAME="OPENAI_API_KEY"
        ;;
      3|deepseek|DeepSeek)
        AI_PROVIDER="deepseek"
        API_KEY="${DEEPSEEK_API_KEY}"
        KEY_NAME="DEEPSEEK_API_KEY"
        ;;
      *)
        AI_PROVIDER="anthropic"
        API_KEY="${ANTHROPIC_API_KEY}"
        KEY_NAME="ANTHROPIC_API_KEY"
        ;;
    esac
    if [ -z "$API_KEY" ]; then
      echo "error: ${KEY_NAME} no está definida o está vacía en .env" >&2
      exit 1
    fi
    if [ -n "$1" ]; then
      ./${OUTPUT_NAME} generate --project "$1" --api-key "${API_KEY}" --ai-provider "${AI_PROVIDER}"
    else
      ./${OUTPUT_NAME} generate --api-key "${API_KEY}" --ai-provider "${AI_PROVIDER}"
    fi
    ;;
esac
