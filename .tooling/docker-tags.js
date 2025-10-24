#!/usr/bin/env node

/**
 * Docker image tagging utilities.
 * Generates multiple tags for semantic versions including major and minor tags.
 */

import { parseSemver } from './image-tags.js';

/**
 * Generate all tags for a given version
 * @param {string} version - The semantic version (e.g., "v2.5.3", "2.5.3")
 * @returns {{full: string, minor: string, major: string, latest: string}|null}
 */
export function generateTags(version) {
  const parsed = parseSemver(version);
  
  if (!parsed) {
    return null;
  }
  
  const hasPrefix = version.startsWith('v');
  const prefix = hasPrefix ? 'v' : '';
  
  return {
    full: `${prefix}${parsed.major}.${parsed.minor}.${parsed.patch}`,
    minor: `${prefix}${parsed.major}.${parsed.minor}`,
    major: `${prefix}${parsed.major}`,
    latest: 'latest',
  };
}

/**
 * Generate Docker image tags with registry prefix
 * @param {string} imageName - Full image name (e.g., "interledger/builders/chartvalidator")
 * @param {string} version - Semantic version
 * @param {string} registry - Registry URL (e.g., "ghcr.io")
 * @returns {string[]} Array of full image tags
 */
export function generateImageTags(imageName, version, registry = 'ghcr.io') {
  const tags = generateTags(version);
  
  if (!tags) {
    throw new Error(`Invalid version format: ${version}`);
  }
  
  const fullImageName = `${registry}/${imageName}`;
  
  return [
    `${fullImageName}:${tags.full}`,
    `${fullImageName}:${tags.minor}`,
    `${fullImageName}:${tags.major}`,
    `${fullImageName}:${tags.latest}`,
  ];
}

/**
 * Generate tag list for docker build-push-action
 * Returns newline-separated list suitable for GitHub Actions
 * @param {string} imageName - Full image name
 * @param {string} version - Semantic version
 * @param {string} registry - Registry URL
 * @returns {string} Newline-separated tag list
 */
export function generateTagList(imageName, version, registry = 'ghcr.io') {
  const tags = generateImageTags(imageName, version, registry);
  return tags.join('\n');
}

/**
 * Format tags for display
 * @param {string} version - Semantic version
 * @returns {string} Formatted tag description
 */
export function formatTagInfo(version) {
  const tags = generateTags(version);
  
  if (!tags) {
    return 'Invalid version';
  }
  
  return `
Tags to be created:
  • ${tags.full} (specific version)
  • ${tags.minor} (minor version - receives patches)
  • ${tags.major} (major version - receives all updates)
  • ${tags.latest} (always latest)

Usage examples:
  docker pull myimage:${tags.full}   # Pin to exact version
  docker pull myimage:${tags.minor}  # Get latest patch for ${tags.major}.${tags.minor.split('.')[1]}
  docker pull myimage:${tags.major}   # Get latest version for major ${tags.major}
  docker pull myimage:latest         # Get latest version
  `.trim();
}

/**
 * Validate that all tags are properly formatted
 * @param {string[]} tags - Array of image tags
 * @returns {{valid: boolean, errors: string[]}}
 */
export function validateTags(tags) {
  const errors = [];
  
  if (!Array.isArray(tags) || tags.length === 0) {
    errors.push('Tags array is empty or not an array');
    return { valid: false, errors };
  }
  
  for (const tag of tags) {
    // Check format: registry/org/image:version
    if (!tag.includes(':')) {
      errors.push(`Tag missing version separator: ${tag}`);
    }
    
    if (!tag.includes('/')) {
      errors.push(`Tag missing registry/org separator: ${tag}`);
    }
  }
  
  return {
    valid: errors.length === 0,
    errors,
  };
}

/**
 * Generate tags for multiple images
 * @param {Array<{name: string, version: string}>} images - Array of image info
 * @param {string} registry - Registry URL
 * @returns {Array<{name: string, version: string, tags: string[]}>}
 */
export function generateTagsForImages(images, registry = 'ghcr.io') {
  return images.map(({ name, version }) => ({
    name,
    version,
    tags: generateImageTags(name, version, registry),
  }));
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  
  if (args.length === 0) {
    console.error('Usage: node docker-tags.js <command> [args]');
    console.error('');
    console.error('Commands:');
    console.error('  generate <version>                    Generate tags for a version');
    console.error('  image <name> <version> [registry]     Generate full image tags');
    console.error('  list <name> <version> [registry]      Generate tag list for docker');
    console.error('  info <version>                        Show tag information');
    console.error('');
    console.error('Examples:');
    console.error('  node docker-tags.js generate v1.2.3');
    console.error('  node docker-tags.js image interledger/builders/chartvalidator v1.2.3');
    console.error('  node docker-tags.js list interledger/builders/chartvalidator v1.2.3 ghcr.io');
    console.error('  node docker-tags.js info v1.2.3');
    process.exit(1);
  }
  
  const command = args[0];
  
  try {
    switch (command) {
      case 'generate': {
        if (args.length < 2) {
          console.error('Usage: node docker-tags.js generate <version>');
          process.exit(1);
        }
        const version = args[1];
        const tags = generateTags(version);
        if (tags) {
          console.log(JSON.stringify(tags, null, 2));
        } else {
          console.error('Invalid version format');
          process.exit(1);
        }
        break;
      }
      
      case 'image': {
        if (args.length < 3) {
          console.error('Usage: node docker-tags.js image <name> <version> [registry]');
          process.exit(1);
        }
        const imageName = args[1];
        const version = args[2];
        const registry = args[3] || 'ghcr.io';
        const tags = generateImageTags(imageName, version, registry);
        tags.forEach(tag => console.log(tag));
        break;
      }
      
      case 'list': {
        if (args.length < 3) {
          console.error('Usage: node docker-tags.js list <name> <version> [registry]');
          process.exit(1);
        }
        const imageName = args[1];
        const version = args[2];
        const registry = args[3] || 'ghcr.io';
        const tagList = generateTagList(imageName, version, registry);
        console.log(tagList);
        break;
      }
      
      case 'info': {
        if (args.length < 2) {
          console.error('Usage: node docker-tags.js info <version>');
          process.exit(1);
        }
        const version = args[1];
        console.log(formatTagInfo(version));
        break;
      }
      
      default:
        console.error(`Unknown command: ${command}`);
        process.exit(1);
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}
