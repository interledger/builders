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
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b /usr/local/bin v2.5.0

# Atlas CLI — generates test migrations for backend/pacioli database schemas
RUN curl -sSf https://atlasgo.sh | sh

WORKDIR /app
