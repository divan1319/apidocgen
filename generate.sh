#!/bin/bash

set -a
source .env 2>/dev/null
set +a

if [ -n "$1" ]; then
  ./${OUTPUT_NAME} generate --project "$1" --api-key "${ANTHROPIC_API_KEY}"
else
  ./${OUTPUT_NAME} generate --api-key "${ANTHROPIC_API_KEY}"
fi
