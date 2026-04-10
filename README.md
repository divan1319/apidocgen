# Generador de Documentación de API con Claude

Este proyecto es una herramienta de generación de documentación de API con Claude. Se necesita una clave de API de Claude para generar la documentación.

## Instalación

1. Clonar el repositorio
2. Ejecutar `build.sh` para construir el binario
3. Ejecutar `./apidocgen` para generar la documentación

## Uso

1. Ejecutar:

    ```bash
    ./apidocgen generate --routes paths/to/routes.php --root /path/to/laravel-project --title "My API Documentation" --api-key ANTHROPIC_API_KEY
    ```

2. La documentación se generará en el archivo `api-docs.html`
3. La documentación se puede abrir en el navegador con el archivo `api-docs.html`

### Frameworks soportados actualmente

- Laravel

### Dependencias

- Go