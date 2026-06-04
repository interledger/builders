# Chart Validator

A CI tool that validates Kubernetes Helm charts end-to-end: it renders each chart with its real values, validates the resulting manifests against Kubernetes API schemas, and confirms that every referenced Docker image actually exists in its registry.

It is designed to run in CI against [ArgoCD ApplicationSet](https://argo-cd.readthedocs.io/en/stable/user-guide/application-set/) files, so problems are caught before a deployment is attempted.

---

## How it works

Chart Validator runs a five-stage pipeline. Each stage is backed by a pool of concurrent workers (10 by default) and passes results to the next stage via Go channels.

```
ApplicationSet YAML files
         │
         ▼
  1. Parse ApplicationSets   — find every chart referenced in each environment
         │
         ▼
  2. Render charts           — helm template with the chart's real values files
         │
         ▼
  3. Validate manifests      — kubeconform against Kubernetes API schemas
         │
         ▼
  4. Extract images          — parse container image references from the manifests
         │
         ▼
  5. Validate images         — docker manifest inspect for every unique image
```

A failure at any stage is reported immediately; the tool exits non-zero if any check fails.

### Stage 1 — Parse ApplicationSets

Scans `{envdir}/{env}/appsets/*.appset.yaml` for entries of the form:

```yaml
spec:
  generators:
    - list:
        elements:
          - chartName: my-chart
            repoURL: https://charts.example.com
            chartVersion: "1.2.3"
            baseValuesFile: env/base/values.yaml
            valuesOverride: env/production/values.yaml
```

Each element becomes one unit of work flowing through the rest of the pipeline. Paths in `baseValuesFile` and `valuesOverride` are relative to the parent directory of `envdir` (i.e. prefixed with `../`).

### Stage 2 — Render charts

Runs `helm template` for each chart, combining the base values file and the override values file. Repository URLs that look like OCI registries but lack a scheme (e.g. `europe-west4-docker.pkg.dev/my-project/charts`) are automatically prefixed with `oci://` because ArgoCD accepts scheme-less URLs but Helm CLI requires the explicit prefix.

Flags passed to Helm:
- `-f baseValuesFile -f valuesOverride` — layered values
- `--version chartVersion`
- `--include-crds`
- `--kube-version` — the target Kubernetes version (defaults to `1.33.0`)
- `--api-versions` — any additional API groups passed via `-api-versions`

Rendered manifests are written to the output directory and passed to the next stage by file path.

### Stage 3 — Validate manifests

Runs `kubeconform` in strict mode against the rendered YAML. CRD schemas are resolved from three sources in order:

1. **Local schemas** — JSON schema files in the `schemas/` directory bundled with the image (covers Traefik, 1Password Connect, Prometheus Operator).
2. **CRDs catalog** — remote fallback at `https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/…` for any CRD not in the local set.
3. **Built-in schemas** — kubeconform's own upstream Kubernetes schemas.

`CustomResourceDefinition` resources themselves are skipped during validation (kubeconform cannot validate a CRD against itself).

### Stage 4 — Extract images

Parses the rendered YAML and collects the `image` field from every `containers` and `initContainers` entry in Pods, Deployments, DaemonSets, and StatefulSets. Duplicate image references are deduplicated before being sent to the next stage.

### Stage 5 — Validate images

Runs `docker manifest inspect {image}` for each unique image. Results are cached so the same image is only checked once even if it appears across multiple charts.

If a manifest inspect fails, the check is retried up to **3 times** with a random backoff of **1–15 seconds** between attempts. Failures that are clearly permanent — image not found (`manifest unknown`), auth errors (`unauthorized`, `denied`), or a bad image reference — are not retried.

---

## Usage

```
chart-checker <command> [flags]

Commands:
  run-checks    Render charts and run all validation checks.
  render-only   Render charts only; skip all validation.
  help          Show this help.
```

### Flags

All flags apply to both commands.

| Flag | Default | Description |
|---|---|---|
| `-env` | _(all environments)_ | Process only this environment (the folder name under `-envdir`). |
| `-envdir` | `../env` | Directory that contains per-environment subdirectories. |
| `-output` | `manifests` | Directory where rendered manifests are written. Cleared on each run. |
| `-api-versions` | _(none)_ | Comma-separated additional Kubernetes API versions passed to `helm template`. |
| `-k8s-version` | `1.33.0` | Kubernetes version used for Helm rendering and kubeconform validation. |
| `-v` | `false` | Enable verbose (debug-level) logging. |

### Examples

```bash
# Validate all environments
chart-checker run-checks

# Validate a single environment
chart-checker run-checks -env sandbox

# Validate with a specific Kubernetes version
chart-checker run-checks -env production -k8s-version 1.30.0

# Render charts only (useful for debugging values or template issues)
chart-checker render-only -env staging -v
```

---

## Registry authentication

Image validation uses the Docker daemon's credential store, so any registry reachable by `docker pull` on the host is also reachable by the validator.

The container image ships with `docker-credential-gcr` pre-configured for Google Artifact Registry endpoints (`gcr.io`, `*.pkg.dev`). For other registries, mount or inject a pre-authenticated `~/.docker/config.json`, or configure the appropriate credential helper in the Docker config before invoking the tool.

---

## Updating CRD schemas

Local CRD schemas must be regenerated whenever a new CRD-bearing chart version is adopted. The schemas live in `schemas/` and are committed to the repository.

From `tmp/ci/`:

```bash
make update-schemas
```

This renders the CRD charts (Traefik, 1Password Connect, Prometheus Operator), converts their OpenAPI specs to JSON Schema format, and writes the output to `schemas/`. To target a specific chart version:

```bash
make update-schemas TRAEFIK_VERSION=35.2.0 PROMETHEUS_OPERATOR_VERSION=76.3.0
```

---

## Building the image

```bash
docker build -t chartvalidator .
```

Unit tests run as part of the build. The resulting image contains `chart-checker`, `helm`, `kubeconform`, `kustomize`, and `docker` CLI.
