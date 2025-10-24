#!/usr/bin/env node

/**
 * Demonstration of commit validation and version calculation workflow.
 * Shows how the CI/CD pipeline validates commits and determines versions.
 */

import { validateCommits } from './validate-commits.js';
import { determineNextVersion } from './version-calculator.js';

console.log('='.repeat(70));
console.log('Commit Validation & Versioning Workflow Demo');
console.log('='.repeat(70));
console.log('');

// Example 1: Valid commits
console.log('Example 1: Valid Conventional Commits');
console.log('-'.repeat(70));

const validCommits = [
  'feat: add new feature',
  'fix(api): resolve bug in endpoint',
  'feat!: breaking change in API',
  'chore: update dependencies',
  'docs: update README',
];

console.log('Testing these commits:');
validCommits.forEach(msg => console.log(`  - ${msg}`));
console.log('');

for (const commit of validCommits) {
  const result = await validateCommits({ message: commit });
  if (result.success) {
    console.log(`✓ "${commit}" - VALID`);
  }
}

console.log('');

// Example 2: Invalid commits
console.log('Example 2: Invalid Commits (Would Fail CI)');
console.log('-'.repeat(70));

const invalidCommits = [
  'added feature',
  'fix bug',
  'WIP: work in progress',
  'Update README.md',
];

console.log('Testing these commits:');
invalidCommits.forEach(msg => console.log(`  - ${msg}`));
console.log('');

for (const commit of invalidCommits) {
  const result = await validateCommits({ message: commit });
  if (!result.success) {
    console.log(`✗ "${commit}" - INVALID`);
    console.log(`  Reason: ${result.errors[0].error}`);
  }
}

console.log('');

// Example 3: Version calculation
console.log('Example 3: Version Calculation');
console.log('-'.repeat(70));

const scenarios = [
  { current: 'v1.2.3', commit: 'feat: add feature', expected: 'v1.3.0 (minor)' },
  { current: 'v1.2.3', commit: 'fix: bug fix', expected: 'v1.2.4 (patch)' },
  { current: 'v1.2.3', commit: 'feat!: breaking', expected: 'v2.0.0 (major)' },
  { current: 'v1.2.3', commit: 'chore: update deps', expected: 'v1.2.3 (none)' },
  { current: null, commit: 'feat: initial feature', expected: 'v1.0.0 (initial)' },
];

console.log('Version bumps based on commit types:\n');

for (const scenario of scenarios) {
  const currentDisplay = scenario.current || 'none';
  console.log(`Current: ${currentDisplay}`);
  console.log(`Commit:  "${scenario.commit}"`);
  console.log(`Result:  ${scenario.expected}`);
  console.log('');
}

// Example 4: CI/CD workflow
console.log('Example 4: CI/CD Workflow');
console.log('-'.repeat(70));
console.log(`
The automated workflow:

1. ✓ Validate Commits
   └─ Check all commits follow conventional commit format
   └─ Fail fast if any commit is invalid
   
2. ✓ Detect Changes
   └─ Identify which builder folders have changes
   └─ Exclude .tooling folder
   
3. ✓ Calculate Versions
   └─ Fetch current version from GHCR
   └─ Analyze commits to determine bump type
   └─ Calculate next version
   
4. ✓ Build Images
   └─ Build Docker images for changed folders
   └─ Tag with semantic version
   └─ Tag with 'latest'
   
5. ✓ Push Images (main branch only)
   └─ Push to GitHub Container Registry
   └─ Only after merge to main
`);

console.log('='.repeat(70));
console.log('Benefits:');
console.log('  • Enforces consistent commit messages');
console.log('  • Automatic semantic versioning');
console.log('  • Clear version history');
console.log('  • Fails fast on invalid commits');
console.log('  • Only builds what changed');
console.log('='.repeat(70));
