# Builders Repository — Agent Instructions

Trust these instructions and only search the codebase if the information below is incomplete or found to be incorrect.

## What This Repository Does

Centralised Docker base images for the Interledger Foundation ecosystem. Two images are currently maintained:

| Folder | Image published at | Purpose |
|---|---|---|
| `chartvalidator/` | `ghcr.io/interledger/builders/chartvalidator` | Helm 3.19, kubeconform 0.7.0, kustomize, docker-cli, docker-credential-gcr, `chart-checker` Go binary |
| `gotester/` | `ghcr.io/interledger/builders/gotester` | Go 1.25, golangci-lint 2.5.0, Atlas CLI 1.2.0, PostgreSQL 17 client, docker-cli, make, bc, bash, git, curl, grep |

## Repository Layout

```
builders/
├── chartvalidator/
│   ├── Dockerfile            # Multi-stage: builds chart-checker from Go, then alpine image
│   ├── README.md
│   └── checker/              # Go source for chart-checker CLI
│       ├── go.mod            # module: github.com/builderslab/chartvalidator/checker, go 1.24.5
│       ├── go.sum
│       ├── main.go           # CLI entry point (run-checks, render-only subcommands)
│       ├── *.go              # Engine files: chart rendering, manifest validation, Docker validation
│       ├── *_test.go         # Unit tests
│       └── test_data/        # YAML fixtures for tests
├── gotester/
│   ├── Dockerfile            # Single-stage: golang:1.25-alpine + tools
│   └── README.md
├── .tooling/                 # Node.js CI helper scripts (excluded from Docker builds)
│   ├── package.json          # type: "module", no external dependencies
│   ├── detect-changes.js     # Which builder folders changed?
│   ├── versioning.js         # Conventional commit parser + SemVer calculator
│   ├── version-calculator.js # Queries GHCR for current version, computes next
│   ├── validate-commits.js   # Commit message linter (used in CI)
│   ├── docker-tags.js        # Tag generation (v1.2.3, v1.2, v1, latest)
│   ├── image-tags.js         # Fetch existing tags from GHCR
│   └── test*.js              # Test suite (no test framework, plain Node.js)
└── .github/
    └── workflows/
        └── buildall.yaml     # Single workflow: validate commits → detect changes → build
```

## Adding a New Builder Image

Create a new top-level directory containing a `Dockerfile`. The CI detects all top-level directories that contain a `Dockerfile` (hidden directories and `.tooling/` are excluded automatically). No other configuration is required.

## Commit Message Format (enforced by CI)

All commits **must** follow Conventional Commits. The CI validates this before anything else runs — a non-conforming commit will fail the pipeline immediately.

```
<type>(<scope>): <description>
```

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`, `build`, `revert`

Version bumps derived from commit type:
- Breaking change (`!` marker or `BREAKING CHANGE:` in body) → major
- `feat` → minor
- `fix`, `perf`, `refactor`, `revert` → patch
- `docs`, `style`, `test`, `chore`, `ci`, `build` → no version bump (image still rebuilt if files changed)

## Building Locally

### Docker images (run from repo root)

```bash
# Build chartvalidator image (runs Go tests inside the Dockerfile)
docker build -t chartvalidator ./chartvalidator

# Build gotester image
docker build -t gotester ./gotester
```

Note: The `chartvalidator` Dockerfile runs `go test .` during the builder stage (`RUN go test .`). If Go tests fail, the Docker build fails.

### Go binary only (no Docker required)

```bash
cd chartvalidator/checker
go test ./...          # Run unit tests — takes ~0.3 s, all pass on a clean checkout
go build -o chart-checker .
```

Always run `go test ./...` from `chartvalidator/checker/`. Running from the repo root fails because there is no `go.mod` there.

## Running Tests

### Go tests (chartvalidator)

```bash
cd chartvalidator/checker
go test ./...
```

Expected output: `ok  github.com/builderslab/chartvalidator/checker  0.260s`

### Node.js tooling tests

`npm` is not required. Run directly with Node.js (≥18):

```bash
cd .tooling
node test.js && node test-image-tags.js && node test-versioning.js && node test-docker-tags.js
```

Expected output: all suites report "0 failed". Individual suites can be run independently. Tests have no external dependencies and run without network access.

## CI Pipeline (`.github/workflows/buildall.yaml`)

Three sequential jobs on every PR and push to `main`:

1. **validate-commits** — runs `node validate-commits.js` from `.tooling/`. Fails if the head commit message is not a valid conventional commit. **This must pass before any Docker build runs.**

2. **detect-changes** — runs `node detect-changes.js detect` from `.tooling/`. Outputs a JSON array of builder folder names that have changed files. On `workflow_dispatch` or when no base ref is available, all folders are built.

3. **build-and-push** — matrix over changed folders:
   - Calls `node version-calculator.js` to fetch the current version from GHCR and compute the next SemVer based on the commit message
   - Builds the Docker image (`docker/build-push-action`)
   - Pushes to `ghcr.io/interledger/builders/<folder>` **only on merge to `main`**; PR builds test that the image builds successfully without pushing

Image tags applied on every release: `v{major}.{minor}.{patch}`, `v{major}.{minor}`, `v{major}`, `latest`.

## Linting and Validation

There is no project-level linter beyond the commit message check. For the Go code in `chartvalidator/checker/`:

```bash
cd chartvalidator/checker
go vet ./...   # No output = pass
go test ./...  # Must pass
```

If `golangci-lint` is available locally: `golangci-lint run ./...` (not enforced in this repo's CI, but the `gotester` image ships it for use in downstream repos).

## Key Facts

- The `.tooling/` scripts use ES module syntax (`import`/`export`); `package.json` sets `"type": "module"`. Do not convert to CommonJS.
- `detect-changes.js` always resolves paths relative to the git repository root regardless of the working directory from which it is invoked.
- Tooling scripts have no `node_modules` and no `package-lock.json` — they have zero external npm dependencies.
- Docker images are pushed only from the `main` branch; PRs only verify that the image builds.
- The `chartvalidator/tmp/` directory is a runtime artefact (gitignored via `chartvalidator/.gitignore`) produced when `chart-checker` is run; do not commit it.
