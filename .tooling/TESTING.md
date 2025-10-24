# Testing Image Tag Fetching

This document provides instructions for testing the image tag fetching functionality with real images.

## Prerequisites

You'll need a GitHub Personal Access Token with `read:packages` permission to fetch tags from GitHub Container Registry.

## Create a Test Token

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate a new token with `read:packages` scope
3. Copy the token

## Testing with Real Images

### Test 1: Fetch tags for the chartvalidator image

```bash
# Export your token
export GITHUB_TOKEN="your_token_here"

# Fetch tags
cd .tooling
node image-tags.js interledger/builders/chartvalidator
```

**Expected output (if image exists):**
```
Fetching tags for: interledger/builders/chartvalidator

Latest version: v1.2.3
All semantic versions (3):
  v1.0.0 (1.0.0)
  v1.1.0 (1.1.0)
  v1.2.3 (1.2.3)
```

**Expected output (if image doesn't exist yet):**
```
Fetching tags for: interledger/builders/chartvalidator

Latest version: none found
All semantic versions (0):
  No semantic version tags found
```

### Test 2: Fetch tags for another public image

You can test with other public images from GHCR:

```bash
# Example with a public image
node image-tags.js cli/cli
```

### Test 3: Use in a script

```javascript
import { getLatestVersion } from './image-tags.js';

const token = process.env.GITHUB_TOKEN;
const imageName = 'interledger/builders/chartvalidator';

const result = await getLatestVersion(imageName, token);

console.log('Latest:', result.latest);
console.log('All versions:', result.all);
```

## Troubleshooting

### 401 Unauthorized
- Check that your token has `read:packages` permission
- Ensure the token is correctly exported as `GITHUB_TOKEN`
- Try with a different public image to verify token works

### 404 Not Found
- The package/image doesn't exist yet in GHCR
- This is expected for new builders that haven't been published yet
- The script will return empty results gracefully

### Rate Limiting
- GitHub API has rate limits
- Authenticated requests have higher limits
- Wait a few minutes if you hit rate limits

## Next Steps

Once this functionality is verified, we'll integrate it into the versioning workflow to:
1. Fetch the latest version tag for an image
2. Parse conventional commits to determine version bump type
3. Calculate and apply the next version
4. Tag the new image with the semantic version
