#!/usr/bin/env node

/**
 * Fetches available tags for Docker images from GitHub Container Registry.
 * This module provides utilities to query and parse semantic version tags.
 */

/**
 * Fetch all tags for a given image from GitHub Container Registry
 * @param {string} imageName - The full image name (e.g., "interledger/builders/chartvalidator")
 * @param {string} [token] - GitHub token for authentication (optional for public images)
 * @returns {Promise<string[]>} Array of tag names
 */
export async function fetchImageTags(imageName, token) {
  // GitHub Container Registry API endpoint
  // Format: https://ghcr.io/v2/{org}/{repo}/tags/list
  const apiUrl = `https://ghcr.io/v2/${imageName}/tags/list`;
  
  const headers = {
    'Accept': 'application/json',
  };
  
  // Add authentication if token is provided
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  try {
    const response = await fetch(apiUrl, { headers });
    
    if (!response.ok) {
      // If unauthorized, try GitHub API instead
      if (response.status === 401 || response.status === 403) {
        return await fetchImageTagsViaGitHubAPI(imageName, token);
      }
      throw new Error(`Failed to fetch tags: ${response.status} ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.tags || [];
  } catch (error) {
    console.error(`Error fetching tags for ${imageName}:`, error.message);
    // Fallback to GitHub API
    return await fetchImageTagsViaGitHubAPI(imageName, token);
  }
}

/**
 * Fetch tags using GitHub Packages API as a fallback
 * @param {string} imageName - The image name (e.g., "interledger/builders/chartvalidator")
 * @param {string} [token] - GitHub token for authentication
 * @returns {Promise<string[]>} Array of tag names
 */
async function fetchImageTagsViaGitHubAPI(imageName, token) {
  // Parse the image name to extract org and package
  // Format: org/repo/package or org/package
  const parts = imageName.split('/');
  if (parts.length < 2) {
    throw new Error(`Invalid image name format: ${imageName}`);
  }
  
  const org = parts[0];
  const packageName = parts.slice(1).join('/');
  
  // GitHub Packages API endpoint
  const apiUrl = `https://api.github.com/orgs/${org}/packages/container/${encodeURIComponent(packageName)}/versions`;
  
  const headers = {
    'Accept': 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  try {
    const response = await fetch(apiUrl, { headers });
    
    if (!response.ok) {
      if (response.status === 404) {
        // Package doesn't exist yet, return empty array
        console.warn(`Package not found: ${imageName}`);
        return [];
      }
      throw new Error(`GitHub API request failed: ${response.status} ${response.statusText}`);
    }
    
    const versions = await response.json();
    
    // Extract all tags from all versions
    const tags = [];
    for (const version of versions) {
      if (version.metadata?.container?.tags) {
        tags.push(...version.metadata.container.tags);
      }
    }
    
    return [...new Set(tags)]; // Remove duplicates
  } catch (error) {
    console.error(`Error fetching tags via GitHub API for ${imageName}:`, error.message);
    return [];
  }
}

/**
 * Parse a semantic version string
 * @param {string} tag - The tag to parse
 * @returns {{major: number, minor: number, patch: number, prerelease: string|null, isValid: boolean}|null}
 */
export function parseSemver(tag) {
  // Match semver pattern: v?1.2.3 or v?1.2.3-prerelease
  const match = tag.match(/^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?$/);
  
  if (!match) {
    return null;
  }
  
  return {
    major: parseInt(match[1], 10),
    minor: parseInt(match[2], 10),
    patch: parseInt(match[3], 10),
    prerelease: match[4] || null,
    isValid: true,
  };
}

/**
 * Filter tags to only semantic version tags
 * @param {string[]} tags - Array of tag names
 * @returns {string[]} Array of semver tags
 */
export function filterSemverTags(tags) {
  return tags.filter(tag => parseSemver(tag) !== null);
}

/**
 * Compare two semantic versions
 * @param {string} a - First version tag
 * @param {string} b - Second version tag
 * @returns {number} -1 if a < b, 0 if a === b, 1 if a > b
 */
export function compareSemver(a, b) {
  const versionA = parseSemver(a);
  const versionB = parseSemver(b);
  
  if (!versionA || !versionB) {
    return 0;
  }
  
  // Compare major version
  if (versionA.major !== versionB.major) {
    return versionA.major - versionB.major;
  }
  
  // Compare minor version
  if (versionA.minor !== versionB.minor) {
    return versionA.minor - versionB.minor;
  }
  
  // Compare patch version
  if (versionA.patch !== versionB.patch) {
    return versionA.patch - versionB.patch;
  }
  
  // If one has prerelease and the other doesn't, stable version is greater
  if (versionA.prerelease && !versionB.prerelease) {
    return -1;
  }
  if (!versionA.prerelease && versionB.prerelease) {
    return 1;
  }
  
  // Both have prerelease or both don't
  if (versionA.prerelease && versionB.prerelease) {
    return versionA.prerelease.localeCompare(versionB.prerelease);
  }
  
  return 0;
}

/**
 * Find the latest semantic version from an array of tags
 * @param {string[]} tags - Array of tag names
 * @param {boolean} [includePrerelease=false] - Whether to include prerelease versions
 * @returns {string|null} The latest semver tag or null if none found
 */
export function findLatestSemver(tags, includePrerelease = false) {
  let semverTags = filterSemverTags(tags);
  
  if (semverTags.length === 0) {
    return null;
  }
  
  // Filter out prerelease versions unless explicitly requested
  if (!includePrerelease) {
    semverTags = semverTags.filter(tag => {
      const parsed = parseSemver(tag);
      return parsed && !parsed.prerelease;
    });
  }
  
  if (semverTags.length === 0) {
    return null;
  }
  
  // Sort tags and return the highest version
  semverTags.sort(compareSemver);
  return semverTags[semverTags.length - 1];
}

/**
 * Get the latest semantic version for an image
 * @param {string} imageName - The full image name
 * @param {string} [token] - GitHub token for authentication
 * @returns {Promise<{latest: string|null, all: string[]}>}
 */
export async function getLatestVersion(imageName, token) {
  const tags = await fetchImageTags(imageName, token);
  const semverTags = filterSemverTags(tags);
  const latest = findLatestSemver(tags);
  
  return {
    latest,
    all: semverTags,
  };
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  
  if (args.length === 0) {
    console.error('Usage: node image-tags.js <image-name> [github-token]');
    console.error('Example: node image-tags.js interledger/builders/chartvalidator');
    process.exit(1);
  }
  
  const imageName = args[0];
  const token = args[1] || process.env.GITHUB_TOKEN;
  
  console.log(`Fetching tags for: ${imageName}\n`);
  
  try {
    const result = await getLatestVersion(imageName, token);
    
    console.log(`Latest version: ${result.latest || 'none found'}`);
    console.log(`\nAll semantic versions (${result.all.length}):`);
    
    if (result.all.length > 0) {
      // Sort and display
      result.all.sort(compareSemver);
      result.all.forEach(tag => {
        const parsed = parseSemver(tag);
        console.log(`  ${tag} (${parsed.major}.${parsed.minor}.${parsed.patch}${parsed.prerelease ? '-' + parsed.prerelease : ''})`);
      });
    } else {
      console.log('  No semantic version tags found');
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}
