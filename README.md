# Generador de Documentación de API con Claude

Este proyecto es una herramienta de generación de documentación de API con Claude. Se necesita una clave de API de Claude para generar la documentación.

## Requisitos

- Claude API key (en el archivo .env)

## Variables de entorno

- ANTHROPIC_API_KEY: API key de Claude
- ROUTES_PATH: Ruta al archivo de rutas
- ROOT_PATH: Ruta al directorio raíz del proyecto
- TITLE: Título de la API
- DOC_LANG: Idioma de la documentación

## Instalación

1. Clonar el repositorio
2. Crear un archivo .env con las variables de entorno del archivo .env.example
3. Ejecutar `build.sh` para construir el binario
4. Ejecutar `./generate.sh` para generar la documentación


## Frameworks soportados actualmente

- Laravel
- Proximamente otros frameworks