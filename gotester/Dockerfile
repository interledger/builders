FROM golang:1.25-alpine

# System dependencies needed by both go-test and mock-tester CI workflows:
#   bash     - shell scripts in CI steps
#   bc       - floating-point coverage threshold comparison
#   curl     - downloading tools (Atlas CLI)
#   git      - Go modules, golangci-lint VCS checks
#   grep     - GNU grep with -P (PCRE) for extracting coverage thresholds
#   make     - mock service Makefiles
#   postgresql17-client - pg_isready to wait for Postgres service containers
RUN apk add --no-cache \
    bash \
    bc \
    curl \
    git \
    grep \
    make \
    postgresql17-client

# golangci-lint v2.5.0 — used by both mock-tester and linting workflows
# Installed from versioned release artifact with checksum verification.
ARG GOLANGCI_LINT_VERSION=2.5.0
ARG GOLANGCI_LINT_SHA256=c77313a77e19b06123962c411d9943cc0d092bbec76b956104d18964e274902e
RUN curl -sSfL "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz" \
      -o /tmp/golangci-lint.tar.gz \
    && echo "${GOLANGCI_LINT_SHA256}  /tmp/golangci-lint.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/golangci-lint.tar.gz -C /tmp \
    && mv /tmp/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64/golangci-lint /usr/local/bin/ \
    && rm -rf /tmp/golangci-lint*

# Atlas CLI — generates test migrations for backend/pacioli database schemas
# Pinned to a specific version via ATLAS_VERSION; bump the ARG to upgrade.
ARG ATLAS_VERSION=v1.2.0
RUN ATLAS_VERSION=${ATLAS_VERSION} curl -sSf https://atlasgo.sh | sh

WORKDIR /app
