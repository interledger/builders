#!/usr/bin/env node

/**
 * Example script demonstrating how to use the image-tags module.
 * This shows how to fetch tags for real images from GHCR.
 */

import { getLatestVersion, fetchImageTags, filterSemverTags } from './image-tags.js';

async function example() {
  console.log('Image Tags Module - Examples\n');
  console.log('='.repeat(60));
  
  // Example 1: Fetch tags for chartvalidator
  console.log('\nExample 1: Fetch tags for chartvalidator');
  console.log('-'.repeat(60));
  
  const imageName = 'interledger/builders/chartvalidator';
  console.log(`Image: ${imageName}`);
  
  try {
    const token = process.env.GITHUB_TOKEN;
    const result = await getLatestVersion(imageName, token);
    
    console.log(`\nLatest version: ${result.latest || '(none found)'}`);
    console.log(`Total semver tags: ${result.all.length}`);
    
    if (result.all.length > 0) {
      console.log('\nAll versions:');
      result.all.forEach(tag => console.log(`  - ${tag}`));
    }
  } catch (error) {
    console.error(`Error: ${error.message}`);
    console.log('\nNote: If the package does not exist yet, this is expected.');
    console.log('Tags will be available after the first image is pushed.');
  }
  
  // Example 2: Working with mock data
  console.log('\n\nExample 2: Working with mock tag data');
  console.log('-'.repeat(60));
  
  const mockTags = [
    'v1.0.0',
    'v1.1.0',
    'v1.2.0',
    'v2.0.0-beta',
    'latest',
    'main-abc123',
    'v0.9.0',
  ];
  
  console.log('Mock tags:', mockTags.join(', '));
  
  const semverOnly = filterSemverTags(mockTags);
  console.log('\nSemantic version tags:', semverOnly.join(', '));
  
  const { findLatestSemver } = await import('./image-tags.js');
  const latest = findLatestSemver(mockTags);
  console.log(`Latest stable version: ${latest}`);
  
  const latestIncludingPre = findLatestSemver(mockTags, true);
  console.log(`Latest (including prerelease): ${latestIncludingPre}`);
  
  // Example 3: Usage in CI/CD
  console.log('\n\nExample 3: Typical CI/CD usage');
  console.log('-'.repeat(60));
  console.log(`
// In your CI/CD script:
import { getLatestVersion } from './image-tags.js';

const imageName = 'interledger/builders/my-builder';
const token = process.env.GITHUB_TOKEN;

const { latest } = await getLatestVersion(imageName, token);

if (latest) {
  console.log(\`Current version: \${latest}\`);
  // Calculate next version based on conventional commit
  const nextVersion = calculateNextVersion(latest, commitMessage);
  console.log(\`Next version: \${nextVersion}\`);
} else {
  console.log('No existing versions, starting with 0.1.0');
}
  `);
  
  console.log('='.repeat(60));
  console.log('\nTo test with your own image:');
  console.log('  node examples.js');
  console.log('\nOr use the CLI directly:');
  console.log('  node image-tags.js <image-name> [token]');
  console.log('\nExample:');
  console.log('  node image-tags.js interledger/builders/chartvalidator');
}

// Run examples
example().catch(error => {
  console.error('Error running examples:', error.message);
  process.exit(1);
});
