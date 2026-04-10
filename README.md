# Generador de Documentación de API

## Instalación

1. Clonar el repositorio
2. Ejecutar `build.sh` para construir el binario
3. Ejecutar `./apidocgen` para generar la documentación

## Uso

1. Ejecutar `./apidocgen` para generar la documentación
    ```bash
    ./apidocgen generate --routes routes/api.php --root /path/to/laravel-project --title "My API Documentation" --api-key ANTHROPIC_API_KEY
    ```
2. La documentación se generará en el archivo `api-docs.html`
3. La documentación se puede abrir en el navegador con el archivo `api-docs.html`