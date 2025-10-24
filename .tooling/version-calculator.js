#!/usr/bin/env node

/**
 * Integration module that combines image tags and versioning.
 * Determines the next version for a Docker image based on commits.
 */

import { getLatestVersion } from './image-tags.js';
import { getNextVersion, getInitialVersion, analyzeManyCommits, calculateNextVersion } from './versioning.js';

/**
 * Determine the next version for an image based on a commit message
 * @param {string} imageName - Full image name (e.g., "interledger/builders/chartvalidator")
 * @param {string} commitMessage - Commit message to analyze
 * @param {string} [token] - GitHub token for authentication
 * @returns {Promise<{current: string|null, next: string, bump: string, isInitial: boolean}>}
 */
export async function determineNextVersion(imageName, commitMessage, token) {
  // Fetch current version from registry
  const { latest } = await getLatestVersion(imageName, token);
  
  if (!latest) {
    // No existing version, calculate initial version
    const initial = getInitialVersion(commitMessage);
    return {
      current: null,
      next: initial,
      bump: 'initial',
      isInitial: true,
    };
  }
  
  // Calculate next version based on commit
  const next = getNextVersion(latest, commitMessage);
  
  // Determine bump type
  let bump = 'none';
  if (next !== latest) {
    const latestParts = latest.replace(/^v/, '').split('.').map(Number);
    const nextParts = next.replace(/^v/, '').split('.').map(Number);
    
    if (nextParts[0] > latestParts[0]) bump = 'major';
    else if (nextParts[1] > latestParts[1]) bump = 'minor';
    else if (nextParts[2] > latestParts[2]) bump = 'patch';
  }
  
  return {
    current: latest,
    next,
    bump,
    isInitial: false,
  };
}

/**
 * Determine versions for multiple images based on their changes
 * @param {Array<{imageName: string, commitMessages: string[]}>} images - Array of images with their commits
 * @param {string} [token] - GitHub token
 * @returns {Promise<Array<{imageName: string, current: string|null, next: string, bump: string}>>}
 */
export async function determineVersionsForImages(images, token) {
  const results = [];
  
  for (const { imageName, commitMessages } of images) {
    // Analyze all commits to determine highest bump needed
    const bumpType = analyzeManyCommits(commitMessages);
    
    // Fetch current version
    const { latest } = await getLatestVersion(imageName, token);
    
    if (!latest) {
      // No existing version, use first commit to determine initial version
      const initial = getInitialVersion(commitMessages[0] || '');
      results.push({
        imageName,
        current: null,
        next: initial,
        bump: 'initial',
      });
    } else if (bumpType !== 'none') {
      // Calculate next version based on bump type
      const next = calculateNextVersion(latest, bumpType);
      results.push({
        imageName,
        current: latest,
        next,
        bump: bumpType,
      });
    } else {
      // No version bump needed
      results.push({
        imageName,
        current: latest,
        next: latest,
        bump: 'none',
      });
    }
  }
  
  return results;
}

/**
 * Format version info for display
 * @param {{current: string|null, next: string, bump: string, isInitial?: boolean}} versionInfo
 * @returns {string}
 */
export function formatVersionInfo(versionInfo) {
  if (versionInfo.isInitial || versionInfo.current === null) {
    return `Initial version: ${versionInfo.next}`;
  }
  
  if (versionInfo.bump === 'none') {
    return `No version change (${versionInfo.current})`;
  }
  
  return `${versionInfo.current} → ${versionInfo.next} (${versionInfo.bump} bump)`;
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  
  if (args.length < 2) {
    console.error('Usage: node version-calculator.js <image-name> <commit-message> [token]');
    console.error('');
    console.error('Examples:');
    console.error('  node version-calculator.js interledger/builders/chartvalidator "feat: add feature"');
    console.error('  node version-calculator.js interledger/builders/chartvalidator "fix: bug" $GITHUB_TOKEN');
    console.error('');
    console.error('The GITHUB_TOKEN environment variable will be used if no token is provided.');
    process.exit(1);
  }
  
  const imageName = args[0];
  const commitMessage = args[1];
  const token = args[2] || process.env.GITHUB_TOKEN;
  
  console.log(`Analyzing version for: ${imageName}`);
  console.log(`Commit: ${commitMessage}\n`);
  
  try {
    const result = await determineNextVersion(imageName, commitMessage, token);
    
    console.log(formatVersionInfo(result));
    console.log('');
    console.log('Details:');
    console.log(`  Current: ${result.current || 'none'}`);
    console.log(`  Next:    ${result.next}`);
    console.log(`  Bump:    ${result.bump}`);
    console.log(`  Initial: ${result.isInitial ? 'yes' : 'no'}`);
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}
