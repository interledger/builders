#!/usr/bin/env node

/**
 * Test suite for image-tags.js
 * Tests semantic version parsing, comparison, and tag filtering
 */

import {
  parseSemver,
  filterSemverTags,
  compareSemver,
  findLatestSemver,
} from './image-tags.js';

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

console.log('Running image-tags tests...\n');

let passed = 0;
let failed = 0;

// Test: parseSemver with various formats
if (test('Parse valid semver: 1.2.3', () => {
  const result = parseSemver('1.2.3');
  assertEquals(result.major, 1);
  assertEquals(result.minor, 2);
  assertEquals(result.patch, 3);
  assertEquals(result.prerelease, null);
  assertEquals(result.isValid, true);
})) passed++; else failed++;

if (test('Parse valid semver with v prefix: v1.2.3', () => {
  const result = parseSemver('v1.2.3');
  assertEquals(result.major, 1);
  assertEquals(result.minor, 2);
  assertEquals(result.patch, 3);
})) passed++; else failed++;

if (test('Parse semver with prerelease: 1.2.3-alpha.1', () => {
  const result = parseSemver('1.2.3-alpha.1');
  assertEquals(result.major, 1);
  assertEquals(result.minor, 2);
  assertEquals(result.patch, 3);
  assertEquals(result.prerelease, 'alpha.1');
})) passed++; else failed++;

if (test('Parse invalid semver returns null', () => {
  assertNull(parseSemver('latest'));
  assertNull(parseSemver('main-abc123'));
  assertNull(parseSemver('1.2'));
  assertNull(parseSemver('not-a-version'));
})) passed++; else failed++;

// Test: filterSemverTags
if (test('Filter semver tags from mixed list', () => {
  const tags = ['v1.0.0', 'latest', 'v1.1.0', 'main-abc123', '2.0.0', 'dev'];
  const filtered = filterSemverTags(tags);
  assertEquals(filtered, ['v1.0.0', 'v1.1.0', '2.0.0']);
})) passed++; else failed++;

if (test('Filter empty list returns empty', () => {
  const filtered = filterSemverTags([]);
  assertEquals(filtered, []);
})) passed++; else failed++;

if (test('Filter list with no semver tags returns empty', () => {
  const tags = ['latest', 'main', 'dev', 'staging'];
  const filtered = filterSemverTags(tags);
  assertEquals(filtered, []);
})) passed++; else failed++;

// Test: compareSemver
if (test('Compare semver: 1.0.0 < 2.0.0 (major)', () => {
  const result = compareSemver('1.0.0', '2.0.0');
  if (result >= 0) throw new Error('Expected negative value');
})) passed++; else failed++;

if (test('Compare semver: 1.1.0 < 1.2.0 (minor)', () => {
  const result = compareSemver('1.1.0', '1.2.0');
  if (result >= 0) throw new Error('Expected negative value');
})) passed++; else failed++;

if (test('Compare semver: 1.0.1 < 1.0.2 (patch)', () => {
  const result = compareSemver('1.0.1', '1.0.2');
  if (result >= 0) throw new Error('Expected negative value');
})) passed++; else failed++;

if (test('Compare semver: 1.0.0 === 1.0.0', () => {
  const result = compareSemver('1.0.0', '1.0.0');
  assertEquals(result, 0);
})) passed++; else failed++;

if (test('Compare semver: 1.0.0-alpha < 1.0.0 (prerelease)', () => {
  const result = compareSemver('1.0.0-alpha', '1.0.0');
  if (result >= 0) throw new Error('Expected negative value');
})) passed++; else failed++;

if (test('Compare semver with v prefix: v1.0.0 < v2.0.0', () => {
  const result = compareSemver('v1.0.0', 'v2.0.0');
  if (result >= 0) throw new Error('Expected negative value');
})) passed++; else failed++;

// Test: findLatestSemver
if (test('Find latest from mixed versions', () => {
  const tags = ['1.0.0', '2.1.0', '1.5.0', 'latest', '2.0.5'];
  const latest = findLatestSemver(tags);
  assertEquals(latest, '2.1.0');
})) passed++; else failed++;

if (test('Find latest with v prefix', () => {
  const tags = ['v1.0.0', 'v2.1.0', 'v1.5.0'];
  const latest = findLatestSemver(tags);
  assertEquals(latest, 'v2.1.0');
})) passed++; else failed++;

if (test('Find latest prefers stable over prerelease', () => {
  const tags = ['1.0.0', '1.0.1-alpha', '0.9.0'];
  const latest = findLatestSemver(tags);
  assertEquals(latest, '1.0.0');
})) passed++; else failed++;

if (test('Find latest returns null for empty list', () => {
  const latest = findLatestSemver([]);
  assertNull(latest);
})) passed++; else failed++;

if (test('Find latest returns null for non-semver tags', () => {
  const tags = ['latest', 'main', 'dev'];
  const latest = findLatestSemver(tags);
  assertNull(latest);
})) passed++; else failed++;

// Test: Complex version sorting
if (test('Sort complex version list correctly', () => {
  const tags = [
    '2.0.0',
    '1.0.0',
    '1.0.1',
    '1.1.0',
    '1.0.0-beta',
    '0.9.9',
    '2.0.1',
    '1.5.3',
  ];
  const sorted = [...tags].sort(compareSemver);
  assertEquals(sorted, [
    '0.9.9',
    '1.0.0-beta',
    '1.0.0',
    '1.0.1',
    '1.1.0',
    '1.5.3',
    '2.0.0',
    '2.0.1',
  ]);
})) passed++; else failed++;

// Test: Edge cases
if (test('Handle versions with large numbers', () => {
  const tags = ['10.20.30', '2.3.4', '100.0.0'];
  const latest = findLatestSemver(tags);
  assertEquals(latest, '100.0.0');
})) passed++; else failed++;

if (test('Parse version with complex prerelease', () => {
  const result = parseSemver('1.0.0-beta.1.2.3');
  assertEquals(result.major, 1);
  assertEquals(result.minor, 0);
  assertEquals(result.patch, 0);
  assertEquals(result.prerelease, 'beta.1.2.3');
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
