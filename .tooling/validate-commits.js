#!/usr/bin/env node

/**
 * Validates commit messages against conventional commit standards.
 * Used in CI/CD to ensure all commits follow the standard.
 */

import { parseConventionalCommit } from './versioning.js';
import { execSync } from 'child_process';

/**
 * Get commit messages from git
 * @param {string} [baseRef] - Base reference to compare against (defaults to HEAD~1)
 * @param {boolean} [onlyLast=false] - Only get the last commit
 * @returns {Array<{sha: string, message: string}>}
 */
function getCommitMessages(baseRef = 'HEAD~1', onlyLast = false) {
  try {
    let command;
    
    if (onlyLast) {
      // Get only the last commit
      command = 'git log -1 --format=%H%n%s%n%b%n---COMMIT---';
    } else {
      // Get commit SHAs and messages in range
      command = `git log ${baseRef}..HEAD --format=%H%n%s%n%b%n---COMMIT---`;
    }
    
    const output = execSync(command, { encoding: 'utf8' });
    
    const commits = [];
    const parts = output.split('---COMMIT---').filter(Boolean);
    
    for (const part of parts) {
      const lines = part.trim().split('\n');
      if (lines.length > 0) {
        const sha = lines[0];
        const message = lines.slice(1).join('\n').trim();
        if (sha && message) {
          commits.push({ sha, message });
        }
      }
    }
    
    return commits;
  } catch (error) {
    console.error('Error fetching commits:', error.message);
    return [];
  }
}

/**
 * Validate a single commit message
 * @param {string} message - Commit message to validate
 * @returns {{valid: boolean, error?: string, skipped?: boolean}}
 */
function validateCommit(message) {
  // Check if it's a merge commit (skip validation)
  if (message.match(/^Merge\s+(branch|pull request|[0-9a-f]+\s+into)/i)) {
    return {
      valid: true,
      skipped: true,
    };
  }
  
  const parsed = parseConventionalCommit(message);
  
  if (!parsed) {
    return {
      valid: false,
      error: 'Does not follow conventional commit format',
    };
  }
  
  // Check for valid types
  const validTypes = [
    'feat', 'fix', 'docs', 'style', 'refactor',
    'perf', 'test', 'chore', 'ci', 'build', 'revert'
  ];
  
  if (!validTypes.includes(parsed.type)) {
    return {
      valid: false,
      error: `Invalid commit type: "${parsed.type}". Valid types: ${validTypes.join(', ')}`,
    };
  }
  
  // Check for meaningful description
  if (parsed.description.length < 3) {
    return {
      valid: false,
      error: 'Description too short (minimum 3 characters)',
    };
  }
  
  return { valid: true };
}

/**
 * Main validation function
 * @param {Object} options
 * @param {string} [options.baseRef] - Base reference for comparison
 * @param {string} [options.message] - Single message to validate (instead of git log)
 * @param {boolean} [options.onlyLast=true] - Only validate the last commit (default: true)
 * @returns {Promise<{success: boolean, errors: Array}>}
 */
export async function validateCommits(options = {}) {
  const errors = [];
  const onlyLast = options.onlyLast !== false; // Default to true
  
  if (options.message) {
    // Validate single message
    const result = validateCommit(options.message);
    if (!result.valid) {
      errors.push({
        message: options.message,
        error: result.error,
      });
    }
  } else {
    // Validate commits from git
    const baseRef = options.baseRef || process.env.BASE_REF || 'HEAD~1';
    const commits = getCommitMessages(baseRef, onlyLast);
    
    if (commits.length === 0) {
      console.warn('No commits found to validate');
      return { success: true, errors: [] };
    }
    
    if (onlyLast) {
      console.log('Validating last commit...\n');
    } else {
      console.log(`Validating ${commits.length} commit(s)...\n`);
    }
    
    for (const commit of commits) {
      const result = validateCommit(commit.message);
      
      if (result.valid) {
        console.log(`✓ ${commit.sha.slice(0, 7)} - ${commit.message.split('\n')[0]}`);
      } else {
        console.error(`✗ ${commit.sha.slice(0, 7)} - ${commit.message.split('\n')[0]}`);
        console.error(`  Error: ${result.error}`);
        errors.push({
          sha: commit.sha,
          message: commit.message,
          error: result.error,
        });
      }
    }
  }
  
  return {
    success: errors.length === 0,
    errors,
  };
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  
  // Check for help flag
  if (args.includes('--help') || args.includes('-h')) {
    console.log('Usage: node validate-commits.js [options]');
    console.log('');
    console.log('Options:');
    console.log('  --base-ref <ref>    Base git reference to compare against (default: HEAD~1)');
    console.log('  --message <msg>     Validate a single message instead of git commits');
    console.log('  --last              Only validate the last commit (default)');
    console.log('  --all               Validate all commits in range');
    console.log('  --help, -h          Show this help message');
    console.log('');
    console.log('Environment Variables:');
    console.log('  BASE_REF            Base reference (used if --base-ref not provided)');
    console.log('');
    console.log('Examples:');
    console.log('  node validate-commits.js                          # Validate last commit');
    console.log('  node validate-commits.js --last                   # Validate last commit (explicit)');
    console.log('  node validate-commits.js --all                    # Validate all commits since base');
    console.log('  node validate-commits.js --all --base-ref origin/main');
    console.log('  node validate-commits.js --message "feat: add feature"');
    console.log('');
    console.log('Exit codes:');
    console.log('  0 - All commits are valid');
    console.log('  1 - One or more commits are invalid');
    process.exit(0);
  }
  
  // Parse arguments
  let baseRef = null;
  let message = null;
  let onlyLast = true; // Default to only validating last commit
  
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--base-ref' && i + 1 < args.length) {
      baseRef = args[i + 1];
      i++;
    } else if (args[i] === '--message' && i + 1 < args.length) {
      message = args[i + 1];
      i++;
    } else if (args[i] === '--last') {
      onlyLast = true;
    } else if (args[i] === '--all') {
      onlyLast = false;
    }
  }
  
  try {
    const result = await validateCommits({ baseRef, message, onlyLast });
    
    console.log('');
    console.log('='.repeat(60));
    
    if (result.success) {
      console.log('✅ All commits are valid!');
      console.log('='.repeat(60));
      process.exit(0);
    } else {
      console.log(`❌ ${result.errors.length} invalid commit(s) found!`);
      console.log('='.repeat(60));
      console.log('');
      console.log('Please ensure all commits follow the Conventional Commits specification:');
      console.log('https://www.conventionalcommits.org/');
      console.log('');
      console.log('Format: <type>(<scope>): <description>');
      console.log('');
      console.log('Valid types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert');
      process.exit(1);
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}
