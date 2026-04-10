#!/bin/bash

# 1. Activamos la exportación automática
set -a 

# 2. Cargamos el archivo .env (asumiendo que ejecutas el script desde la raíz)
source .env

# 3. Ejecutamos el comando
./apidocgen generate \
--routes "$ROUTES_PATH" \
--root "$ROOT_PATH" \
--title "$TITLE" \
--api-key "$ANTHROPIC_API_KEY" \
--doc-lang "$DOC_LANG"
