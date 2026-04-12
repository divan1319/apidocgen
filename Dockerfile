# syntax=docker/dockerfile:1

# Etapa 1: frontend Vue (salida en web/dist)
FROM node:20-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Etapa 2: binario Go con assets embebidos (embed.go → web/dist)
FROM golang:1.22-alpine AS backend
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY embed.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY --from=frontend /src/web/dist ./web/dist
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/apidocgen ./cmd/main.go

# Etapa para extraer el binario al host: docker build --target artifact -o . .
FROM scratch AS artifact
COPY --from=backend /out/apidocgen /apidocgen

# Imagen por defecto (última etapa): ejecutar apidocgen en contenedor
FROM alpine:3.20 AS runtime
RUN apk --no-cache add ca-certificates tzdata
COPY --from=backend /out/apidocgen /usr/local/bin/apidocgen
ENTRYPOINT ["/usr/local/bin/apidocgen"]
CMD ["help"]
