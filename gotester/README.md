# Go Tester

Pre-built image for running Go unit tests, mock-service tests, and linting in the [interledger-app](https://github.com/interledger/interledger-app) CI pipeline.

## What's inside

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25 | Compile and test Go code |
| golangci-lint | 2.5.0 | Lint all Go packages |
| Atlas CLI | 1.2.0 | Generate database test migrations |
| Docker CLI | — | Mock e2e tests that spin up containers (e.g. Redis) |
| PostgreSQL 17 client | 17.x | `pg_isready` to wait for Postgres service containers |
| GNU grep (PCRE) | — | Extract coverage thresholds with `-oP` |
| bc | — | Floating-point coverage comparison |
| make | — | Run mock-service Makefiles |
| bash, git, curl | — | Standard CI utilities |

## Replaces

This single image replaces the tool-installation steps in three interledger-app workflow templates:

- **`go-test-template.yml`** — `setup-go`, `Install Atlas CLI`, `pg_isready` wait loop
- **`mock-tester.yml`** — `setup-go`, `golangci-lint-action`, Docker setup
- **`linting.yml`** — `setup-go`, `golangci-lint-action`

## Usage

```yaml
# GitHub Actions example
jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/interledger/builders/gotester:latest
    services:
      postgres:
        image: postgres:17
    steps:
      - uses: actions/checkout@v4
      - run: go test -coverprofile=coverage.out ./go/backend/...
```

## Building locally

```bash
cd gotester
docker build -t gotester .
```
