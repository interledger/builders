#!/usr/bin/env node

/**
 * Conventional commit parser and version calculator.
 * Parses conventional commit messages and determines semantic version bumps.
 * 
 * Based on Conventional Commits specification:
 * https://www.conventionalcommits.org/
 */

/**
 * Parse a conventional commit message
 * @param {string} message - The commit message to parse
 * @returns {{type: string, scope: string|null, breaking: boolean, description: string, isValid: boolean}|null}
 */
export function parseConventionalCommit(message) {
  if (!message || typeof message !== 'string') {
    return null;
  }
  
  // Extract first line only
  const firstLine = message.split('\n')[0].trim();
  
  // Pattern: type(scope)?!?: description
  // Examples:
  //   feat: add new feature
  //   feat(api)!: breaking change
  //   fix(core): fix bug
  const pattern = /^([a-z]+)(?:\(([^)]+)\))?(!)?:\s*(.+)$/i;
  const match = firstLine.match(pattern);
  
  if (!match) {
    return null;
  }
  
  const [, type, scope, breakingMarker, description] = match;
  
  // Check for BREAKING CHANGE in body
  const hasBreakingInBody = /^BREAKING[- ]CHANGE:/mi.test(message);
  const breaking = !!breakingMarker || hasBreakingInBody;
  
  return {
    type: type.toLowerCase(),
    scope: scope || null,
    breaking,
    description: description.trim(),
    isValid: true,
  };
}

/**
 * Determine the version bump type from a conventional commit
 * @param {string} message - The commit message
 * @param {boolean} [strict=false] - If true, throw error for non-conventional commits
 * @returns {'major'|'minor'|'patch'|'none'}
 */
export function getVersionBump(message, strict = false) {
  const parsed = parseConventionalCommit(message);
  
  if (!parsed) {
    if (strict) {
      throw new Error(`Commit message does not follow conventional commit format: "${message}"`);
    }
    // Non-conventional commits default to patch
    return 'patch';
  }
  
  // Breaking change = major bump
  if (parsed.breaking) {
    return 'major';
  }
  
  // Features = minor bump
  if (parsed.type === 'feat' || parsed.type === 'feature') {
    return 'minor';
  }
  
  // Fixes and other changes = patch bump
  if (parsed.type === 'fix' || 
      parsed.type === 'perf' || 
      parsed.type === 'refactor' ||
      parsed.type === 'revert') {
    return 'patch';
  }
  
  // Non-production changes = no bump
  if (parsed.type === 'docs' ||
      parsed.type === 'style' ||
      parsed.type === 'test' ||
      parsed.type === 'chore' ||
      parsed.type === 'ci' ||
      parsed.type === 'build') {
    return 'none';
  }
  
  // Unknown types default to patch for safety
  return 'patch';
}

/**
 * Calculate the next version based on current version and bump type
 * @param {string} currentVersion - Current version (e.g., "1.2.3" or "v1.2.3")
 * @param {'major'|'minor'|'patch'|'none'} bumpType - Type of version bump
 * @param {boolean} [keepPrefix=true] - Keep the 'v' prefix if present in current version
 * @returns {string} Next version
 */
export function calculateNextVersion(currentVersion, bumpType, keepPrefix = true) {
  // Parse current version
  const hasPrefix = currentVersion.startsWith('v');
  const versionString = hasPrefix ? currentVersion.slice(1) : currentVersion;
  
  const match = versionString.match(/^(\d+)\.(\d+)\.(\d+)(?:-.*)?$/);
  if (!match) {
    throw new Error(`Invalid version format: ${currentVersion}`);
  }
  
  let [, major, minor, patch] = match;
  major = parseInt(major, 10);
  minor = parseInt(minor, 10);
  patch = parseInt(patch, 10);
  
  // Apply bump
  let nextVersion;
  switch (bumpType) {
    case 'major':
      nextVersion = `${major + 1}.0.0`;
      break;
    case 'minor':
      nextVersion = `${major}.${minor + 1}.0`;
      break;
    case 'patch':
      nextVersion = `${major}.${minor}.${patch + 1}`;
      break;
    case 'none':
      nextVersion = `${major}.${minor}.${patch}`;
      break;
    default:
      throw new Error(`Invalid bump type: ${bumpType}`);
  }
  
  // Add prefix back if it was present and should be kept
  return (hasPrefix && keepPrefix) ? `v${nextVersion}` : nextVersion;
}

/**
 * Calculate next version from commit message
 * @param {string} currentVersion - Current version
 * @param {string} commitMessage - Commit message to parse
 * @param {boolean} [keepPrefix=true] - Keep version prefix
 * @param {boolean} [strict=false] - If true, throw error for non-conventional commits
 * @returns {string} Next version
 */
export function getNextVersion(currentVersion, commitMessage, keepPrefix = true, strict = false) {
  const bumpType = getVersionBump(commitMessage, strict);
  return calculateNextVersion(currentVersion, bumpType, keepPrefix);
}

/**
 * Analyze multiple commits and determine the highest bump type needed
 * @param {string[]} messages - Array of commit messages
 * @param {boolean} [strict=false] - If true, throw error for non-conventional commits
 * @returns {'major'|'minor'|'patch'|'none'}
 */
export function analyzeManyCommits(messages, strict = false) {
  if (!messages || messages.length === 0) {
    return 'none';
  }
  
  let hasMajor = false;
  let hasMinor = false;
  let hasPatch = false;
  
  for (const message of messages) {
    const bump = getVersionBump(message, strict);
    
    if (bump === 'major') {
      hasMajor = true;
      break; // Major is highest, can stop early
    }
    if (bump === 'minor') {
      hasMinor = true;
    }
    if (bump === 'patch') {
      hasPatch = true;
    }
  }
  
  if (hasMajor) return 'major';
  if (hasMinor) return 'minor';
  if (hasPatch) return 'patch';
  return 'none';
}

/**
 * Get initial version when no version exists
 * @param {string} commitMessage - First commit message
 * @returns {string} Initial version
 */
export function getInitialVersion(commitMessage = '') {
  const bump = getVersionBump(commitMessage);
  
  // For breaking changes or features, start at 1.0.0
  if (bump === 'major' || bump === 'minor') {
    return 'v1.0.0';
  }
  
  // For other changes, start at 0.1.0
  return 'v0.1.0';
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  
  if (args.length === 0) {
    console.error('Usage: node versioning.js <command> [args]');
    console.error('');
    console.error('Commands:');
    console.error('  parse <message>              Parse a conventional commit');
    console.error('  validate <message>           Validate a conventional commit (exits 1 if invalid)');
    console.error('  bump <message>               Get bump type for a commit');
    console.error('  next <current> <message>     Calculate next version');
    console.error('  initial [message]            Get initial version');
    console.error('');
    console.error('Examples:');
    console.error('  node versioning.js parse "feat: add new feature"');
    console.error('  node versioning.js validate "fix: bug fix"');
    console.error('  node versioning.js bump "fix: bug fix"');
    console.error('  node versioning.js next v1.2.3 "feat!: breaking change"');
    console.error('  node versioning.js initial "feat: first feature"');
    process.exit(1);
  }
  
  const command = args[0];
  
  try {
    switch (command) {
      case 'parse': {
        const message = args.slice(1).join(' ');
        const parsed = parseConventionalCommit(message);
        if (parsed) {
          console.log(JSON.stringify(parsed, null, 2));
        } else {
          console.log('Not a valid conventional commit');
          process.exit(1);
        }
        break;
      }
      
      case 'validate': {
        const message = args.slice(1).join(' ');
        const parsed = parseConventionalCommit(message);
        if (parsed) {
          console.log('✓ Valid conventional commit');
          console.log(`  Type: ${parsed.type}`);
          if (parsed.scope) console.log(`  Scope: ${parsed.scope}`);
          if (parsed.breaking) console.log(`  Breaking: yes`);
          console.log(`  Description: ${parsed.description}`);
          process.exit(0);
        } else {
          console.error('✗ Invalid conventional commit format');
          console.error('');
          console.error('Expected format: <type>(<scope>): <description>');
          console.error('');
          console.error('Valid types: feat, fix, docs, style, refactor, perf, test, chore, ci, build');
          console.error('');
          console.error('Examples:');
          console.error('  feat: add new feature');
          console.error('  fix(api): fix bug in endpoint');
          console.error('  feat!: breaking change');
          console.error('');
          console.error('See: https://www.conventionalcommits.org/');
          process.exit(1);
        }
        break;
      }
      
      case 'bump': {
        const message = args.slice(1).join(' ');
        const bump = getVersionBump(message);
        console.log(bump);
        break;
      }
      
      case 'next': {
        if (args.length < 3) {
          console.error('Usage: node versioning.js next <current-version> <commit-message>');
          process.exit(1);
        }
        const currentVersion = args[1];
        const message = args.slice(2).join(' ');
        const nextVersion = getNextVersion(currentVersion, message);
        console.log(nextVersion);
        break;
      }
      
      case 'initial': {
        const message = args.slice(1).join(' ');
        const version = getInitialVersion(message);
        console.log(version);
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
