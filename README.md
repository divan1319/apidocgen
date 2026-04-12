# apidocgen — Documentación de APIs con IA

Herramienta en Go que analiza las rutas de tu proyecto (Laravel, .NET, etc.), envía los endpoints a la API de **Anthropic (Claude)** y genera documentación HTML lista para abrir en el navegador. Incluye interfaz web para gestionar proyectos y una CLI para automatizar la generación.

## Características

- Parsers para **Laravel** y **.NET** (`dotnet` en flags y configuración).
- Salida HTML en `docs/` con caché opcional en `cache/` para no repetir llamadas a la API.
- Documentación generada en **inglés** o **español** (`en` / `es`).
- **CLI** (`generate`, `serve`) y **panel web** al ejecutar `serve`.
- Concurrencia configurable (`--workers`) para acelerar la generación.

## Requisitos

- [Go](https://go.dev/dl/) **1.22** o superior (ver `go.mod`).
- [Node.js](https://nodejs.org/) **18+** y npm (para compilar el frontend en `web/`).
- Cuenta y **clave de API de Anthropic** ([Claude](https://www.anthropic.com/)).

En el futuro se podrán añadir más proveedores de IA además de Anthropic.

## Instalación

1. Clona el repositorio y entra en el directorio del proyecto.
2. Copia la plantilla de entorno y rellena la clave:
   - `cp .env.example .env`
   - Edita `.env` y asigna `ANTHROPIC_API_KEY`.
3. Haz ejecutables los scripts si hace falta: `chmod +x build.sh generate.sh`.
4. Compila todo (frontend Vue + binario Go):
   - `./build.sh`  
   El binario queda como **`./apidocgen`** en la raíz del repo.

### Variables en `.env`

| Variable | Descripción |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Clave de API de Anthropic (obligatoria para generar o servir). |
| `OUTPUT_NAME` | Nombre del binario que usará `generate.sh`. Debe coincidir con el archivo generado por `build.sh` (por defecto **`apidocgen`**). |
| `PORT` | Puerto HTTP cuando eliges la opción de servidor en `generate.sh` (por defecto `8080`). |

## Uso

### Script `generate.sh`

Carga `.env` y ofrece un menú:

1. **Generar documentación (CLI)** — ejecuta el binario con `generate`.
2. **Iniciar servidor web** — ejecuta `serve` en `http://localhost:${PORT}`.

Si llamas al script con un argumento en el modo generar, se pasa como `--project` (slug del proyecto):

```bash
./generate.sh mi-api-laravel
```

### CLI directa

```bash
./apidocgen help
./apidocgen generate
./apidocgen generate --project mi-slug
./apidocgen serve --port 8080
```

**Generar** sin flags abre un asistente para elegir o crear un proyecto (framework, archivos de rutas, raíz del código, idioma de la documentación, etc.). Los proyectos se guardan en `projects/<slug>.json`.

**Servir** expone la UI y los HTML generados bajo `/docs/`.

### Flags útiles de `generate`

| Flag | Descripción |
|------|-------------|
| `--lang` | Framework: `laravel`, `dotnet` (por defecto `laravel`). |
| `--routes` | Archivos de rutas separados por comas (p. ej. `routes/api.php`). |
| `--root` | Raíz del proyecto a documentar (por defecto `.`). |
| `--output` | Ruta del HTML (por defecto `docs/<slug>.html`). |
| `--title` | Título que aparece en la documentación. |
| `--doc-lang` | `en` o `es`. |
| `--api-key` | Clave API (si no usas `ANTHROPIC_API_KEY`). |
| `--cache` | Archivo de caché o `none` para desactivarla. |
| `--force` | Ignorar caché y volver a generar todo con Claude. |
| `--workers` | Peticiones concurrentes a la API (por defecto `5`). |

## Estructura de datos local

Tras generar documentación, suelen aparecer:

- `projects/` — definición de cada proyecto (rutas, framework, idioma, etc.).
- `docs/` — HTML generado (`<slug>.html`).
- `cache/` — caché por proyecto para reducir coste y tiempo.

## Frameworks soportados

- **Laravel** — `--lang laravel`
- **.NET** — `--lang dotnet`

Se pueden ampliar con nuevos parsers registrados en el código.

## Licencia

Uso restringido: permitido el uso personal e interno; no se permite la distribución ni la redistribución del software. Consulta [LICENSE.md](LICENSE.md) para el texto completo.
