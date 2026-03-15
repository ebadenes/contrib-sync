# contrib-sync

`contrib-sync` mirrors contribution timestamps from Gitea into a GitHub mirror repository using empty commits, so your GitHub contribution graph reflects work done outside GitHub.

This project is a new implementation in Go. The idea is inspired by `greens`, and it also builds on a previous personal Python script.

## Features

- Collect commits from Gitea repositories
- Optionally collect pull requests, issues, and reviews
- Normalize activity into timestamped contribution events
- Create empty commits in a GitHub mirror repository
- Run entirely with Docker, without installing Go locally

## Project Status

This repository is in early development. The initial scaffold is in place and the full sync flow is being implemented incrementally.

## Planned Commands

- `contrib-sync init`
- `contrib-sync sync`
- `contrib-sync status`
- `contrib-sync version`

## Development

### Requirements

- Docker
- A Gitea personal access token
- A GitHub mirror repository already created locally or available to clone

### Build

```bash
make build
```

If you already have a local Go image and want to avoid pulling a new one:

```bash
make build GO_IMAGE=golang:1.24
```

### Test

```bash
make test
```

### Run

Create your local `config.yaml` from `config.example.yaml`, then run:

```bash
make run
```

## Configuration

The project uses a YAML configuration file. See `config.example.yaml` for the expected structure.

## License

GPL-3.0-only
