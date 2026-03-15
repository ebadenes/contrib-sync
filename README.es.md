# contrib-sync

`contrib-sync` refleja marcas temporales de contribuciones desde Gitea hacia un repositorio espejo en GitHub usando commits vacios, para que el grafico de contribuciones de GitHub muestre trabajo realizado fuera de GitHub.

Este proyecto es una implementacion nueva en Go. La idea esta inspirada en `greens` y tambien parte de un script personal previo en Python.

## Caracteristicas

- Recoge commits desde repositorios de Gitea
- Opcionalmente recoge pull requests, issues y reviews
- Normaliza la actividad en eventos con marca temporal
- Crea commits vacios en un repositorio espejo de GitHub
- Se ejecuta completamente con Docker, sin instalar Go en local

## Estado del proyecto

Este repositorio esta en una fase inicial. El scaffold base ya existe y el flujo completo de sincronizacion se implementara de forma incremental.

## Comandos previstos

- `contrib-sync init`
- `contrib-sync sync`
- `contrib-sync status`
- `contrib-sync version`

## Desarrollo

### Requisitos

- Docker
- Un token personal de Gitea
- Un repositorio espejo en GitHub ya creado en local o disponible para clonar

### Compilar

```bash
make build
```

Si ya tienes una imagen local de Go y quieres evitar descargar otra:

```bash
make build GO_IMAGE=golang:1.24
```

### Testear

```bash
make test
```

### Ejecutar

Crea tu `config.yaml` local a partir de `config.example.yaml` y luego ejecuta:

```bash
make run
```

## Configuracion

El proyecto usa un archivo de configuracion YAML. Consulta `config.example.yaml` para ver la estructura esperada.

## Licencia

GPL-3.0-only
