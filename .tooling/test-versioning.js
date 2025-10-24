#!/usr/bin/env node

/**
 * Test suite for versioning.js
 * Tests conventional commit parsing and version calculation
 */

import {
  parseConventionalCommit,
  getVersionBump,
  calculateNextVersion,
  getNextVersion,
  analyzeManyCommits,
  getInitialVersion,
} from './versioning.js';

function test(name, fn) {
  try {
    fn();
    console.log(`✓ ${name}`);
    return true;
  } catch (error) {
    console.error(`✗ ${name}`);
    console.error(`  ${error.message}`);
    return false;
  }
}

function assertEquals(actual, expected, message) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(
      message || `Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`
    );
  }
}

function assertNull(actual, message) {
  if (actual !== null) {
    throw new Error(message || `Expected null, got ${JSON.stringify(actual)}`);
  }
}

console.log('Running versioning tests...\n');

let passed = 0;
let failed = 0;

// Test: parseConventionalCommit
if (test('Parse simple feat commit', () => {
  const result = parseConventionalCommit('feat: add new feature');
  assertEquals(result.type, 'feat');
  assertEquals(result.scope, null);
  assertEquals(result.breaking, false);
  assertEquals(result.description, 'add new feature');
})) passed++; else failed++;

if (test('Parse feat with scope', () => {
  const result = parseConventionalCommit('feat(api): add endpoint');
  assertEquals(result.type, 'feat');
  assertEquals(result.scope, 'api');
  assertEquals(result.breaking, false);
})) passed++; else failed++;

if (test('Parse breaking change with !', () => {
  const result = parseConventionalCommit('feat!: breaking change');
  assertEquals(result.type, 'feat');
  assertEquals(result.breaking, true);
})) passed++; else failed++;

if (test('Parse breaking change with scope and !', () => {
  const result = parseConventionalCommit('feat(api)!: breaking change');
  assertEquals(result.type, 'feat');
  assertEquals(result.scope, 'api');
  assertEquals(result.breaking, true);
})) passed++; else failed++;

if (test('Parse breaking change from body', () => {
  const message = 'feat: new feature\n\nBREAKING CHANGE: this breaks stuff';
  const result = parseConventionalCommit(message);
  assertEquals(result.breaking, true);
})) passed++; else failed++;

if (test('Parse fix commit', () => {
  const result = parseConventionalCommit('fix: fix bug');
  assertEquals(result.type, 'fix');
  assertEquals(result.breaking, false);
})) passed++; else failed++;

if (test('Parse chore commit', () => {
  const result = parseConventionalCommit('chore: update dependencies');
  assertEquals(result.type, 'chore');
})) passed++; else failed++;

if (test('Invalid commit returns null', () => {
  assertNull(parseConventionalCommit('not a conventional commit'));
})) passed++; else failed++;

if (test('Empty string returns null', () => {
  assertNull(parseConventionalCommit(''));
})) passed++; else failed++;

// Test: getVersionBump
if (test('Breaking change = major bump', () => {
  assertEquals(getVersionBump('feat!: breaking'), 'major');
  assertEquals(getVersionBump('fix!: breaking'), 'major');
})) passed++; else failed++;

if (test('feat = minor bump', () => {
  assertEquals(getVersionBump('feat: new feature'), 'minor');
})) passed++; else failed++;

if (test('fix = patch bump', () => {
  assertEquals(getVersionBump('fix: bug fix'), 'patch');
})) passed++; else failed++;

if (test('perf = patch bump', () => {
  assertEquals(getVersionBump('perf: optimize code'), 'patch');
})) passed++; else failed++;

if (test('refactor = patch bump', () => {
  assertEquals(getVersionBump('refactor: clean code'), 'patch');
})) passed++; else failed++;

if (test('chore = no bump', () => {
  assertEquals(getVersionBump('chore: update deps'), 'none');
})) passed++; else failed++;

if (test('docs = no bump', () => {
  assertEquals(getVersionBump('docs: update readme'), 'none');
})) passed++; else failed++;

if (test('ci = no bump', () => {
  assertEquals(getVersionBump('ci: update workflow'), 'none');
})) passed++; else failed++;

if (test('Non-conventional = patch bump', () => {
  assertEquals(getVersionBump('random commit message'), 'patch');
})) passed++; else failed++;

// Test: calculateNextVersion
if (test('Major bump from 1.2.3', () => {
  assertEquals(calculateNextVersion('1.2.3', 'major', false), '2.0.0');
})) passed++; else failed++;

if (test('Minor bump from 1.2.3', () => {
  assertEquals(calculateNextVersion('1.2.3', 'minor', false), '1.3.0');
})) passed++; else failed++;

if (test('Patch bump from 1.2.3', () => {
  assertEquals(calculateNextVersion('1.2.3', 'patch', false), '1.2.4');
})) passed++; else failed++;

if (test('No bump from 1.2.3', () => {
  assertEquals(calculateNextVersion('1.2.3', 'none', false), '1.2.3');
})) passed++; else failed++;

if (test('Keep v prefix when present', () => {
  assertEquals(calculateNextVersion('v1.2.3', 'minor', true), 'v1.3.0');
})) passed++; else failed++;

if (test('Remove v prefix when requested', () => {
  assertEquals(calculateNextVersion('v1.2.3', 'minor', false), '1.3.0');
})) passed++; else failed++;

if (test('Handle version without prefix', () => {
  assertEquals(calculateNextVersion('1.2.3', 'patch', true), '1.2.4');
})) passed++; else failed++;

if (test('Handle prerelease version', () => {
  assertEquals(calculateNextVersion('1.2.3-alpha', 'patch', false), '1.2.4');
})) passed++; else failed++;

// Test: getNextVersion
if (test('Get next version from feat commit', () => {
  assertEquals(getNextVersion('v1.2.3', 'feat: new feature'), 'v1.3.0');
})) passed++; else failed++;

if (test('Get next version from fix commit', () => {
  assertEquals(getNextVersion('v1.2.3', 'fix: bug fix'), 'v1.2.4');
})) passed++; else failed++;

if (test('Get next version from breaking change', () => {
  assertEquals(getNextVersion('v1.2.3', 'feat!: breaking'), 'v2.0.0');
})) passed++; else failed++;

if (test('Get next version from chore (no bump)', () => {
  assertEquals(getNextVersion('v1.2.3', 'chore: update'), 'v1.2.3');
})) passed++; else failed++;

// Test: analyzeManyCommits
if (test('Multiple commits: highest is major', () => {
  const commits = [
    'fix: bug fix',
    'feat!: breaking change',
    'feat: new feature',
  ];
  assertEquals(analyzeManyCommits(commits), 'major');
})) passed++; else failed++;

if (test('Multiple commits: highest is minor', () => {
  const commits = [
    'fix: bug fix',
    'feat: new feature',
    'chore: update',
  ];
  assertEquals(analyzeManyCommits(commits), 'minor');
})) passed++; else failed++;

if (test('Multiple commits: highest is patch', () => {
  const commits = [
    'fix: bug fix',
    'perf: optimize',
    'chore: update',
  ];
  assertEquals(analyzeManyCommits(commits), 'patch');
})) passed++; else failed++;

if (test('Multiple commits: all non-production', () => {
  const commits = [
    'chore: update',
    'docs: update readme',
    'ci: update workflow',
  ];
  assertEquals(analyzeManyCommits(commits), 'none');
})) passed++; else failed++;

if (test('Empty commits array', () => {
  assertEquals(analyzeManyCommits([]), 'none');
})) passed++; else failed++;

// Test: getInitialVersion
if (test('Initial version for feat', () => {
  assertEquals(getInitialVersion('feat: first feature'), 'v1.0.0');
})) passed++; else failed++;

if (test('Initial version for breaking change', () => {
  assertEquals(getInitialVersion('feat!: breaking'), 'v1.0.0');
})) passed++; else failed++;

if (test('Initial version for fix', () => {
  assertEquals(getInitialVersion('fix: bug fix'), 'v0.1.0');
})) passed++; else failed++;

if (test('Initial version for chore', () => {
  assertEquals(getInitialVersion('chore: setup'), 'v0.1.0');
})) passed++; else failed++;

if (test('Initial version with no message', () => {
  assertEquals(getInitialVersion(), 'v0.1.0');
})) passed++; else failed++;

// Test: Edge cases
if (test('Handle multiline commit message', () => {
  const message = 'feat: new feature\n\nThis is a longer description\nwith multiple lines';
  assertEquals(getVersionBump(message), 'minor');
})) passed++; else failed++;

if (test('Handle commit with extra whitespace', () => {
  const result = parseConventionalCommit('  feat:   add feature  ');
  assertEquals(result.type, 'feat');
  assertEquals(result.description, 'add feature');
})) passed++; else failed++;

if (test('Case insensitive type matching', () => {
  const result = parseConventionalCommit('FEAT: new feature');
  assertEquals(result.type, 'feat');
})) passed++; else failed++;

console.log(`\n${'='.repeat(50)}`);
console.log(`Results: ${passed} passed, ${failed} failed`);
console.log(`${'='.repeat(50)}\n`);

if (failed > 0) {
  console.log('❌ Some tests failed!');
  process.exit(1);
} else {
  console.log('✅ All tests passed!');
  process.exit(0);
}
