# Builders Tooling

This directory contains scripts and utilities used by GitHub Actions workflows for the builders repository.

## Scripts

### detect-changes.js

Detects which builder folders have changes and should be rebuilt.

**Usage:**

```bash
# Detect changed folders (requires git context and BASE_REF env var)
node detect-changes.js detect

# List all buildable folders
node detect-changes.js list
```

**Environment Variables:**

- `GITHUB_EVENT_NAME`: The GitHub event that triggered the workflow
- `BASE_REF`: The base commit reference to compare against

**Output:**

Returns a JSON array of folder names to stdout. Diagnostic messages are written to stderr.

**Behavior:**

- In `list` mode: Returns all folders containing a Dockerfile (excludes hidden folders and `.tooling`)
- In `detect` mode: Returns only folders with changes between BASE_REF and HEAD
- On `workflow_dispatch` events: Returns all buildable folders
- If no changes detected: Returns all buildable folders (safety fallback)

## Testing

Run the test suite to verify the tooling works correctly:

```bash
npm test
# or
node test.js
```

Tests verify:
- Script can list all buildable folders
- `.tooling` folder is excluded from builds
- Hidden folders (starting with `.`) are excluded
- Output is valid JSON

## Design Principles

Following the pattern used in the `charts` repository:

1. **Keep CI logic in scripts**: Complex logic lives in JavaScript files, not in YAML
2. **Easy to test**: Scripts can be run locally without GitHub Actions
3. **Clear separation**: Build/deploy logic separate from change detection
4. **Fail-safe**: When in doubt, build everything rather than miss changes
