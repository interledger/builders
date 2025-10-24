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

### versioning.js

Parses conventional commits and calculates semantic version bumps.

**Usage:**

```bash
# Parse a commit message
node versioning.js parse "feat: add new feature"

# Validate a commit message (exits 1 if invalid)
node versioning.js validate "fix: bug fix"

# Get bump type
node versioning.js bump "feat!: breaking change"

# Calculate next version
node versioning.js next v1.2.3 "feat: add feature"

# Get initial version for new project
node versioning.js initial "feat: first feature"
```

**As a module:**

```javascript
import { parseConventionalCommit, getVersionBump, getNextVersion } from './versioning.js';

// Parse commit
const parsed = parseConventionalCommit('feat(api): add endpoint');
// { type: 'feat', scope: 'api', breaking: false, description: 'add endpoint', isValid: true }

// Get bump type (with strict mode)
const bump = getVersionBump('feat: new feature', true); // throws if invalid
// 'minor'

// Calculate next version
const next = getNextVersion('v1.2.3', 'fix: bug fix');
// 'v1.2.4'
```

### validate-commits.js

Validates commit messages in CI/CD pipelines to ensure conventional commit compliance.

**Usage:**

```bash
# Validate commits since base ref
node validate-commits.js --base-ref origin/main

# Validate single message
node validate-commits.js --message "feat: add feature"

# Use environment variable
BASE_REF=origin/main node validate-commits.js
```

**Exit codes:**
- `0`: All commits are valid
- `1`: One or more commits are invalid (pipeline will fail)

**Integration in CI:**

The workflow automatically validates all commits in PRs and pushes. If any commit doesn't follow conventional commit format, the build will fail before any Docker images are built.

### image-tags.js

Fetches and analyzes Docker image tags from GitHub Container Registry.

**Usage:**

```bash
# Fetch tags for an image
node image-tags.js <image-name> [github-token]

# Example
node image-tags.js interledger/builders/chartvalidator

# With authentication
node image-tags.js interledger/builders/chartvalidator $GITHUB_TOKEN
```

**As a module:**

```javascript
import { getLatestVersion, parseSemver, findLatestSemver } from './image-tags.js';

// Fetch latest version
const { latest, all } = await getLatestVersion('interledger/builders/my-image', token);
console.log(`Latest: ${latest}`); // e.g., "1.2.3"
console.log(`All versions: ${all}`); // e.g., ["1.0.0", "1.1.0", "1.2.3"]

// Parse semantic version
const parsed = parseSemver('v1.2.3-alpha');
// { major: 1, minor: 2, patch: 3, prerelease: 'alpha', isValid: true }

// Find latest from tag list
const tags = ['v1.0.0', 'v2.1.0', 'latest', 'v1.5.0'];
const latest = findLatestSemver(tags);
// "v2.1.0"
```

**Functions:**

- `fetchImageTags(imageName, token)` - Fetch all tags for an image
- `parseSemver(tag)` - Parse a semantic version string
- `filterSemverTags(tags)` - Filter array to only semver tags
- `compareSemver(a, b)` - Compare two semantic versions
- `findLatestSemver(tags, includePrerelease)` - Find the latest version
- `getLatestVersion(imageName, token)` - Get latest version and all semver tags

## Testing

Run the test suite to verify the tooling works correctly:

```bash
npm test
# or
node test.js && node test-image-tags.js

# Run specific test suites
npm run test:changes    # Test change detection
npm run test:tags       # Test image tag functionality
```

**Tests verify:**
- Script can list all buildable folders
- `.tooling` folder is excluded from builds
- Hidden folders (starting with `.`) are excluded
- Output is valid JSON
- Semantic version parsing works correctly
- Version comparison and sorting works
- Latest version detection works

## Examples

See `examples.js` for practical usage examples:

```bash
node examples.js
```

## Design Principles

Following the pattern used in the `charts` repository:

1. **Keep CI logic in scripts**: Complex logic lives in JavaScript files, not in YAML
2. **Easy to test**: Scripts can be run locally without GitHub Actions
3. **Clear separation**: Build/deploy logic separate from change detection
4. **Fail-safe**: When in doubt, build everything rather than miss changes
