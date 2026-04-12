# apidocgen

Herramienta en Go que analiza las rutas de tu proyecto (Laravel, .NET, etc.), envía los endpoints a la API de **Anthropic (Claude)** y genera documentación HTML lista para abrir en el navegador.

Incluye dos modos de uso: una **CLI interactiva** para generar documentación desde la terminal, y un **servidor web** con panel de administración para gestionar múltiples proyectos desde el navegador.

## Características

- Parsers para **Laravel** y **.NET**.
- Soporte **multi-proyecto**: cada proyecto se guarda como un archivo JSON en `projects/` con su propia configuración.
- **Caché por proyecto** en `cache/` para no repetir llamadas a la API.
- Documentación generada en **inglés** o **español**.
- **CLI** (`generate`) con menú interactivo para seleccionar o crear proyectos.
- **Servidor web** (`serve`) con frontend Vue.js embebido en el binario.
- Portal `docs/index.html` que lista todas las documentaciones generadas.
- Concurrencia configurable (`--workers`) para acelerar la generación.

## Requisitos

- [Go](https://go.dev/dl/) **1.22** o superior.
- [Node.js](https://nodejs.org/) **18+** y npm (para compilar el frontend).
- Clave de API de **Anthropic** ([Claude](https://www.anthropic.com/)).

## Instalación

```bash
git clone <repo-url>
cd apidocgen
cp .env.example .env
```

Edita `.env` y configura las variables:


| Variable            | Descripción                                                           |
| ------------------- | --------------------------------------------------------------------- |
| `ANTHROPIC_API_KEY` | Clave de API de Anthropic (obligatoria).                              |
| `OUTPUT_NAME`       | Nombre del binario generado por `build.sh` (por defecto `apidocgen`). |
| `PORT`              | Puerto HTTP para el servidor web (por defecto `8080`).                |


Compila el frontend y el binario:

```bash
chmod +x build.sh generate.sh
./build.sh
```

El script `build.sh` hace lo siguiente:

1. Instala dependencias y compila el frontend Vue (`web/dist/`).
2. Compila el binario Go con el frontend embebido dentro.

## Uso

### Script `generate.sh`

La forma más sencilla de usar la herramienta. Carga `.env` y muestra un menú:

```
¿Qué deseas hacer?
  [1] Generar documentación (CLI)
  [2] Iniciar servidor web
  Selecciona [1/2] (default: 1):
```

**Opción 1 — CLI**: lanza el flujo interactivo donde puedes seleccionar un proyecto existente o crear uno nuevo. Si pasas un argumento, se usa como slug del proyecto:

```bash
./generate.sh               # menú interactivo
./generate.sh mi-api         # selecciona proyecto "mi-api" directamente
```

**Opción 2 — Servidor web**: inicia el panel en `http://localhost:PORT` donde puedes crear, editar, eliminar proyectos y lanzar generaciones desde el navegador.

### CLI directa

```bash
./apidocgen help                              # ver ayuda
./apidocgen generate                          # menú interactivo
./apidocgen generate --project mi-api         # generar proyecto específico
./apidocgen serve --port 8080                 # iniciar servidor web
```

### Flags de `generate`


| Flag         | Descripción                                            |
| ------------ | ------------------------------------------------------ |
| `--project`  | Slug del proyecto (salta el menú interactivo).         |
| `--lang`     | Framework: `laravel`, `dotnet` (default: `laravel`).   |
| `--routes`   | Archivos de rutas separados por coma.                  |
| `--root`     | Directorio raíz del proyecto a documentar.             |
| `--output`   | Ruta del HTML de salida (default: `docs/<slug>.html`). |
| `--title`    | Título de la documentación.                            |
| `--doc-lang` | Idioma: `en` o `es`.                                   |
| `--api-key`  | Clave API (alternativa a `ANTHROPIC_API_KEY`).         |
| `--cache`    | Archivo de caché, o `none` para desactivar.            |
| `--force`    | Ignorar caché y regenerar todo.                        |
| `--workers`  | Peticiones concurrentes a Claude (default: `5`).       |


### Flags de `serve`


| Flag        | Descripción                                    |
| ----------- | ---------------------------------------------- |
| `--port`    | Puerto HTTP (default: `8080`).                 |
| `--api-key` | Clave API (alternativa a `ANTHROPIC_API_KEY`). |


## Estructura del proyecto

```
apidocgen/
├── cmd/main.go                  # Punto de entrada (generate + serve)
├── embed.go                     # go:embed del frontend compilado
├── internal/
│   ├── ai/                      # Cliente de Anthropic (Claude)
│   ├── cache/                   # Sistema de caché por proyecto
│   ├── generator/               # Generador HTML (docs + index)
│   ├── parser/                  # Parsers de rutas
│   │   ├── laravel/
│   │   └── dotnet/
│   ├── project/                 # CRUD de proyectos (JSON)
│   └── server/                  # Servidor HTTP + API REST + motor de generación
├── pkg/models/                  # Modelos compartidos
├── web/                         # Frontend Vue 3 + Tailwind CSS v4
│   ├── src/
│   │   ├── views/               # Dashboard, ProjectForm, DocsViewer
│   │   ├── components/          # Navbar, ProjectCard
│   │   └── api/                 # Cliente HTTP para la API REST
│   └── dist/                    # Build compilado (embebido en el binario)
├── projects/                    # Configuración de cada proyecto (.json)
├── cache/                       # Caché de documentación por proyecto
├── docs/                        # HTML generado + index.html
├── build.sh                     # Compilar frontend + binario
└── generate.sh                  # Script de ejecución rápida
```

## Estructura de datos local


| Carpeta     | Contenido                                                                      | Se versiona |
| ----------- | ------------------------------------------------------------------------------ | ----------- |
| `projects/` | Un `.json` por proyecto con su configuración (lang, routes, root, title, etc.) | Sí          |
| `docs/`     | HTML generado por proyecto + `index.html` como portal de entrada               | Sí          |
| `cache/`    | Caché de respuestas de Claude por proyecto                                     | Si          |


## API REST (servidor web)

Cuando se ejecuta `serve`, se exponen estos endpoints:


| Método   | Ruta                           | Descripción                                |
| -------- | ------------------------------ | ------------------------------------------ |
| `GET`    | `/api/projects`                | Listar todos los proyectos                 |
| `GET`    | `/api/projects/:slug`          | Obtener un proyecto                        |
| `POST`   | `/api/projects`                | Crear proyecto                             |
| `PUT`    | `/api/projects/:slug`          | Actualizar proyecto                        |
| `DELETE` | `/api/projects/:slug`          | Eliminar proyecto y sus archivos           |
| `POST`   | `/api/projects/:slug/generate` | Lanzar generación de documentación         |
| `GET`    | `/api/docs/:slug`              | Verificar si existe documentación generada |
| `GET`    | `/api/settings`                | Parsers e idiomas disponibles              |


Los HTML generados se sirven en `/docs/<slug>.html`.

## Frameworks soportados

- **Laravel** — `--lang laravel`
- **.NET / ASP.NET Core** — `--lang dotnet`

Se pueden agregar nuevos parsers implementando la interfaz `Parser` y registrándolos en `internal/parser/`.

## Ejemplos de configuración de rutas

El campo **routes** indica qué archivos debe analizar el parser para encontrar los endpoints. Es relativo al **root** del proyecto.

### Laravel

En Laravel las rutas se definen en archivos PHP dentro de `routes/`. Puedes apuntar a uno o varios archivos separados por coma.

**Un solo archivo de rutas:**

```
routes: routes/api.php
root:   /home/user/mi-proyecto-laravel
```

Esto analiza `/home/user/mi-proyecto-laravel/routes/api.php`. Si ese archivo tiene `include` o `require` a otros archivos (por ejemplo rutas versionadas), el parser los resuelve automáticamente.

**Múltiples archivos:**

```
routes: routes/api.php,routes/web.php
root:   /home/user/mi-proyecto-laravel
```

**Rutas versionadas con includes:**

Si tu `routes/api.php` tiene algo como:

```php
<?php
require __DIR__.'/api/v1/users.php';
require __DIR__.'/api/v1/products.php';
require __DIR__.'/api/v2/users.php';
```

Basta con apuntar a `routes/api.php` y el parser seguirá los includes automáticamente, generando una sección por cada archivo.

**Apuntar directamente a un archivo de versión:**

```
routes: routes/api/v1/users.php
root:   /home/user/mi-proyecto-laravel
```

### .NET / ASP.NET Core

En .NET las rutas se definen dentro de controladores (clases con atributos `[Route]`, `[HttpGet]`, etc.) o en `Program.cs` con Minimal APIs.

**Carpeta de controladores:**

```
routes: Controllers/
root:   /home/user/mi-proyecto-dotnet
```

Esto escanea recursivamente todos los archivos `.cs` dentro de `Controllers/` buscando endpoints.

**Controlador específico:**

```
routes: Controllers/UsersController.cs
root:   /home/user/mi-proyecto-dotnet
```

**Múltiples fuentes (controladores + Minimal APIs):**

```
routes: Controllers/,Program.cs
root:   /home/user/mi-proyecto-dotnet
```

Esto analiza todos los controladores en `Controllers/` y también los endpoints definidos con `app.MapGet(...)`, `app.MapPost(...)`, etc. en `Program.cs`.

**Solo una carpeta de versión:**

```
routes: Controllers/Api/V1/
root:   /home/user/mi-proyecto-dotnet
```

### Notas generales

- Las rutas son **relativas al root** del proyecto. No necesitas poner la ruta absoluta completa.
- Si pones una **carpeta** (solo .NET), el parser escanea recursivamente todos los `.cs` dentro.
- Puedes separar **múltiples archivos o carpetas** con coma: `routes/api.php,routes/admin.php`.
- Cada archivo procesado genera una **sección** independiente en la documentación final.

## Licencia

Uso restringido: permitido el uso personal e interno; no se permite la distribución ni la redistribución del software. Consulta [LICENSE.md](LICENSE.md) para el texto completo.