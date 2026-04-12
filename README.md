# apidocgen

Herramienta en Go que analiza las rutas de tu proyecto (Laravel, .NET, Node.js, etc.), envía los endpoints a la API de **Anthropic (Claude)** y genera documentación HTML lista para abrir en el navegador.

Incluye dos modos de uso: una **CLI interactiva** para generar documentación desde la terminal, y un **servidor web** con panel de administración para gestionar múltiples proyectos desde el navegador.

## Características

- Parsers para **Laravel**, **.NET**, **Express**, **Fastify** y **Node.js HTTP nativo**.
- Soporte **multi-proyecto**: cada proyecto se guarda como un archivo JSON en `projects/` con su propia configuración.
- **Caché por proyecto** en `cache/` para no repetir llamadas a la API.
- Documentación generada en **inglés** o **español**.
- **CLI** (`generate`) con menú interactivo para seleccionar o crear proyectos.
- **Servidor web** (`serve`) con frontend Vue.js embebido en el binario.
- Portal `docs/index.html` que lista todas las documentaciones generadas.
- Concurrencia configurable (`--workers`) para acelerar la generación.

## Requisitos

- Clave de API de **Anthropic** ([Claude](https://www.anthropic.com/)).
- Para **compilar en tu máquina** (sin Docker):
  - [Go](https://go.dev/dl/) **1.22** o superior.
  - [Node.js](https://nodejs.org/) **18+** y npm (para compilar el frontend).
- Para **compilar o ejecutar solo con Docker**: [Docker](https://docs.docker.com/get-docker/) con BuildKit (habilitado por defecto en versiones recientes).

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

Si no tienes Go ni Node instalados, usa la variante con Docker (ver [Docker](#docker)):

```bash
./build.sh --docker
```

## Docker

Puedes construir **solo con Docker** (multi-stage: Node para el frontend, Go para el binario) o **ejecutar** la herramienta dentro de un contenedor Alpine con certificados CA para llamadas HTTPS a Anthropic.

### Compilar el binario en el host (`./apidocgen`)

Equivale a un `build.sh` local, pero las herramientas corren dentro de contenedores. El resultado es `./apidocgen` en la raíz del repositorio:

```bash
./build.sh --docker
```

Requisitos: solo Docker (BuildKit). Internamente se usa `docker build --target artifact -o .`.

### Construir la imagen para ejecutar

La última etapa del `Dockerfile` es una imagen lista para `docker run` (binario en `/usr/local/bin/apidocgen`):

```bash
docker build -t apidocgen .
```

### Ejecutar el servidor web (`serve`)

La aplicación usa rutas relativas (`projects/`, `cache/`, `docs/`). Conviene fijar un directorio de trabajo en el contenedor y montar esas carpetas desde el host para no perder datos al parar el contenedor.

```bash
mkdir -p projects cache docs

docker run --rm -it \
  -w /data \
  -v "$(pwd)/projects:/data/projects" \
  -v "$(pwd)/cache:/data/cache" \
  -v "$(pwd)/docs:/data/docs" \
  -p 8080:8080 \
  -e ANTHROPIC_API_KEY="tu-clave" \
  apidocgen serve --port 8080
```

Abre `http://localhost:8080` en el navegador.

### Ejecutar la CLI (`generate`)

Para el modo interactivo necesitas TTY (`-it`). Los mismos volúmenes guardan proyectos, caché y HTML generados.

```bash
docker run --rm -it \
  -w /data \
  -v "$(pwd)/projects:/data/projects" \
  -v "$(pwd)/cache:/data/cache" \
  -v "$(pwd)/docs:/data/docs" \
  -e ANTHROPIC_API_KEY="tu-clave" \
  apidocgen generate
```

El campo **root** de cada proyecto (ruta al código que quieres documentar) debe ser una ruta **visible dentro del contenedor**. Suele montarse el repositorio o carpeta del API como volumen adicional, por ejemplo:

```bash
docker run --rm -it \
  -w /data \
  -v "$(pwd)/projects:/data/projects" \
  -v "$(pwd)/cache:/data/cache" \
  -v "$(pwd)/docs:/data/docs" \
  -v /ruta/en/tu/host/mi-api-laravel:/source \
  -e ANTHROPIC_API_KEY="tu-clave" \
  apidocgen generate --project mi-slug
```

En la configuración del proyecto, usa `root` apuntando a esa ruta interna (p. ej. `/source`) o ajusta el montaje para que coincida con lo que guardaste en el JSON del proyecto.

### Ayuda y otros comandos

```bash
docker run --rm apidocgen help
docker run --rm apidocgen serve --help
docker run --rm apidocgen generate --help
```

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
| `--lang`     | Framework: `laravel`, `dotnet`, `express`, `fastify`, `nodehttp` (default: `laravel`). |
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
│   │   ├── dotnet/
│   │   └── node/                # Express, Fastify y Node HTTP nativo
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
├── Dockerfile                   # Build multi-stage + imagen de ejecución
├── .dockerignore                # Contexto reducido para docker build
├── build.sh                     # Compilar frontend + binario (o --docker)
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
- **Express** — `--lang express`
- **Fastify** — `--lang fastify`
- **Node.js HTTP nativo** — `--lang nodehttp`

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

### Express (`--lang express`)

Parser para aplicaciones Express. Detecta rutas directas, Router con prefijo, route chaining y middleware.

**Carpeta de rutas:**

```
routes: routes/
root:   /home/user/mi-api-express
lang:   express
```

**Archivo principal:**

```
routes: src/app.js
root:   /home/user/mi-api-express
lang:   express
```

**Múltiples archivos:**

```
routes: src/routes/users.ts,src/routes/products.ts,src/app.ts
root:   /home/user/mi-api-express
lang:   express
```

#### Patrones que detecta

**Rutas directas:**

```javascript
app.get('/api/users', getUsers);
app.post('/api/users', createUser);
app.put('/api/users/:id', updateUser);
app.delete('/api/users/:id', deleteUser);
```

**Router con prefijo:**

```javascript
const usersRouter = express.Router();
usersRouter.get('/', getUsers);
usersRouter.get('/:id', getUser);
usersRouter.post('/', createUser);

app.use('/api/users', usersRouter);
// Resultado: GET /api/users, GET /api/users/:id, POST /api/users
```

Cuando el `Router()` y su `app.use()` están en el mismo archivo, el parser resuelve el prefijo automáticamente.

**Route chaining:**

```javascript
app.route('/api/books')
    .get(getBooks)
    .post(createBook);

app.route('/api/books/:id')
    .get(getBook)
    .put(updateBook)
    .delete(deleteBook);
```

**Middleware:**

```javascript
app.get('/api/admin', authenticate, authorize, (req, res) => {
    res.json({ admin: true });
});
// Detecta middleware: [authenticate, authorize]
```

#### Ejemplo CLI

```bash
./apidocgen generate \
    --lang express \
    --routes src/routes/ \
    --root /home/user/mi-api-express \
    --title "Mi API Express" \
    --doc-lang es
```

### Fastify (`--lang fastify`)

Parser para aplicaciones Fastify. Detecta rutas shorthand, route objects, register con prefijo y patrón plugin.

**Carpeta de rutas:**

```
routes: src/routes/
root:   /home/user/mi-api-fastify
lang:   fastify
```

#### Patrones que detecta

**Rutas shorthand:**

```javascript
fastify.get('/api/users', getUsers);
fastify.post('/api/users', createUser);
fastify.put('/api/users/:id', updateUser);
fastify.delete('/api/users/:id', deleteUser);
```

**Route object (declaración completa):**

```javascript
fastify.route({
    method: 'POST',
    url: '/api/users',
    preHandler: [authenticate, validate],
    handler: createUser
});

// También soporta arrays de métodos:
fastify.route({
    method: ['GET', 'HEAD'],
    url: '/api/health',
    handler: healthCheck
});
```

**Register con prefijo:**

```javascript
fastify.register(userRoutes, { prefix: '/api/users' });
fastify.register(itemRoutes, { prefix: '/api/items' });
```

**Patrón plugin:**

```javascript
module.exports = async function(fastify, opts) {
    fastify.get('/', getUsers);
    fastify.get('/:id', getUser);
    fastify.post('/', createUser);
};
```

#### Ejemplo CLI

```bash
./apidocgen generate \
    --lang fastify \
    --routes src/app.ts \
    --root /home/user/mi-api-fastify \
    --title "API Fastify"
```

### Node.js HTTP nativo (`--lang nodehttp`)

Parser para servidores HTTP nativos de Node.js. Detecta patrones con `req.method`/`req.url` y bloques `switch/case`.

**Archivo del servidor:**

```
routes: server.js
root:   /home/user/mi-servidor-node
lang:   nodehttp
```

#### Patrones que detecta

**if/else con req.method y req.url:**

```javascript
const server = http.createServer((req, res) => {
    if (req.method === 'GET' && req.url === '/api/users') {
        getUsers(req, res);
    } else if (req.method === 'POST' && req.url === '/api/users') {
        createUser(req, res);
    }
});
```

**switch/case:**

```javascript
const server = http.createServer((req, res) => {
    if (req.url === '/api/users') {
        switch (req.method) {
            case 'GET': getUsers(req, res); break;
            case 'POST': createUser(req, res); break;
        }
    }
});
```

#### Ejemplo CLI

```bash
./apidocgen generate \
    --lang nodehttp \
    --routes server.js,routes/api.js \
    --root /home/user/mi-servidor-node \
    --title "API HTTP Nativa"
```

#### Notas comunes (Express, Fastify, Node HTTP)

- Se procesan archivos `.js`, `.ts`, `.mjs`, `.mts`, `.cjs` y `.cts`.
- Los archivos TypeScript se marcan con language `typescript` para code fences correctos en la documentación.
- Se extraen automáticamente los **parámetros de ruta** (`:id`, `:userId`, etc.) como parámetros requeridos.
- Los tres parsers ignoran código de testing (`describe`, `test`, `it`, `supertest`, etc.).
- Se omiten automáticamente carpetas `node_modules`, `dist`, `build`, `coverage` y `__tests__`.

### Notas generales

- Las rutas son **relativas al root** del proyecto. No necesitas poner la ruta absoluta completa.
- Si pones una **carpeta** (.NET, Express, Fastify o Node HTTP), el parser escanea recursivamente los archivos correspondientes.
- Puedes separar **múltiples archivos o carpetas** con coma: `routes/api.php,routes/admin.php`.
- Cada archivo procesado genera una **sección** independiente en la documentación final.

## Licencia

Uso restringido: permitido el uso personal e interno; no se permite la distribución ni la redistribución del software. Consulta [LICENSE.md](LICENSE.md) para el texto completo.