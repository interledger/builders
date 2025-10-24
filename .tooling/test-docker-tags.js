#!/usr/bin/env node

/**
 * Test suite for docker-tags.js
 * Tests tag generation for Docker images
 */

import {
  generateTags,
  generateImageTags,
  generateTagList,
  validateTags,
  generateTagsForImages,
} from './docker-tags.js';

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

console.log('Running docker-tags tests...\n');

let passed = 0;
let failed = 0;

// Test: generateTags
if (test('Generate tags for v1.2.3', () => {
  const tags = generateTags('v1.2.3');
  assertEquals(tags.full, 'v1.2.3');
  assertEquals(tags.minor, 'v1.2');
  assertEquals(tags.major, 'v1');
  assertEquals(tags.latest, 'latest');
})) passed++; else failed++;

if (test('Generate tags for 2.5.0 (no prefix)', () => {
  const tags = generateTags('2.5.0');
  assertEquals(tags.full, '2.5.0');
  assertEquals(tags.minor, '2.5');
  assertEquals(tags.major, '2');
  assertEquals(tags.latest, 'latest');
})) passed++; else failed++;

if (test('Generate tags for v10.20.30', () => {
  const tags = generateTags('v10.20.30');
  assertEquals(tags.full, 'v10.20.30');
  assertEquals(tags.minor, 'v10.20');
  assertEquals(tags.major, 'v10');
})) passed++; else failed++;

if (test('Invalid version returns null', () => {
  assertNull(generateTags('latest'));
  assertNull(generateTags('not-a-version'));
})) passed++; else failed++;

// Test: generateImageTags
if (test('Generate image tags with registry', () => {
  const tags = generateImageTags('interledger/builders/app', 'v1.2.3', 'ghcr.io');
  assertEquals(tags, [
    'ghcr.io/interledger/builders/app:v1.2.3',
    'ghcr.io/interledger/builders/app:v1.2',
    'ghcr.io/interledger/builders/app:v1',
    'ghcr.io/interledger/builders/app:latest',
  ]);
})) passed++; else failed++;

if (test('Generate image tags with default registry', () => {
  const tags = generateImageTags('org/image', 'v2.0.0');
  assertEquals(tags.length, 4);
  assertEquals(tags[0], 'ghcr.io/org/image:v2.0.0');
  assertEquals(tags[1], 'ghcr.io/org/image:v2.0');
  assertEquals(tags[2], 'ghcr.io/org/image:v2');
  assertEquals(tags[3], 'ghcr.io/org/image:latest');
})) passed++; else failed++;

if (test('Generate image tags for version without v prefix', () => {
  const tags = generateImageTags('myorg/myapp', '3.4.5', 'docker.io');
  assertEquals(tags, [
    'docker.io/myorg/myapp:3.4.5',
    'docker.io/myorg/myapp:3.4',
    'docker.io/myorg/myapp:3',
    'docker.io/myorg/myapp:latest',
  ]);
})) passed++; else failed++;

// Test: generateTagList
if (test('Generate newline-separated tag list', () => {
  const list = generateTagList('org/app', 'v1.0.0', 'ghcr.io');
  const lines = list.split('\n');
  assertEquals(lines.length, 4);
  assertEquals(lines[0], 'ghcr.io/org/app:v1.0.0');
  assertEquals(lines[1], 'ghcr.io/org/app:v1.0');
  assertEquals(lines[2], 'ghcr.io/org/app:v1');
  assertEquals(lines[3], 'ghcr.io/org/app:latest');
})) passed++; else failed++;

// Test: validateTags
if (test('Validate correct tags', () => {
  const tags = [
    'ghcr.io/org/app:v1.0.0',
    'ghcr.io/org/app:v1.0',
    'ghcr.io/org/app:v1',
    'ghcr.io/org/app:latest',
  ];
  const result = validateTags(tags);
  assertEquals(result.valid, true);
  assertEquals(result.errors.length, 0);
})) passed++; else failed++;

if (test('Detect missing version separator', () => {
  const tags = ['ghcr.io/org/app'];
  const result = validateTags(tags);
  assertEquals(result.valid, false);
  if (!result.errors[0].includes('version separator')) {
    throw new Error('Should detect missing version separator');
  }
})) passed++; else failed++;

if (test('Detect missing registry separator', () => {
  const tags = ['myapp:v1.0.0'];
  const result = validateTags(tags);
  assertEquals(result.valid, false);
  if (!result.errors[0].includes('registry/org separator')) {
    throw new Error('Should detect missing registry/org separator');
  }
})) passed++; else failed++;

if (test('Reject empty tag array', () => {
  const result = validateTags([]);
  assertEquals(result.valid, false);
})) passed++; else failed++;

// Test: generateTagsForImages
if (test('Generate tags for multiple images', () => {
  const images = [
    { name: 'org/app1', version: 'v1.0.0' },
    { name: 'org/app2', version: 'v2.5.3' },
  ];
  const results = generateTagsForImages(images);
  
  assertEquals(results.length, 2);
  assertEquals(results[0].name, 'org/app1');
  assertEquals(results[0].version, 'v1.0.0');
  assertEquals(results[0].tags.length, 4);
  
  assertEquals(results[1].name, 'org/app2');
  assertEquals(results[1].version, 'v2.5.3');
  assertEquals(results[1].tags.length, 4);
})) passed++; else failed++;

// Test: Edge cases
if (test('Handle v0.1.0 version', () => {
  const tags = generateTags('v0.1.0');
  assertEquals(tags.major, 'v0');
  assertEquals(tags.minor, 'v0.1');
  assertEquals(tags.full, 'v0.1.0');
})) passed++; else failed++;

if (test('Handle large version numbers', () => {
  const tags = generateTags('v100.200.300');
  assertEquals(tags.major, 'v100');
  assertEquals(tags.minor, 'v100.200');
  assertEquals(tags.full, 'v100.200.300');
})) passed++; else failed++;

if (test('Tags always include latest', () => {
  const tags1 = generateTags('v1.0.0');
  const tags2 = generateTags('v2.0.0');
  const tags3 = generateTags('v0.1.0');
  
  assertEquals(tags1.latest, 'latest');
  assertEquals(tags2.latest, 'latest');
  assertEquals(tags3.latest, 'latest');
})) passed++; else failed++;

// Test: Practical scenarios
if (test('Scenario: First release v1.0.0', () => {
  const tags = generateImageTags('interledger/builders/app', 'v1.0.0', 'ghcr.io');
  
  // All tags point to v1.0.0
  assertEquals(tags[0], 'ghcr.io/interledger/builders/app:v1.0.0');
  assertEquals(tags[1], 'ghcr.io/interledger/builders/app:v1.0');
  assertEquals(tags[2], 'ghcr.io/interledger/builders/app:v1');
  assertEquals(tags[3], 'ghcr.io/interledger/builders/app:latest');
})) passed++; else failed++;

if (test('Scenario: Patch release v1.0.1', () => {
  const tags = generateImageTags('interledger/builders/app', 'v1.0.1', 'ghcr.io');
  
  // v1.0 and v1 tags get updated, v1.0.0 stays
  assertEquals(tags[0], 'ghcr.io/interledger/builders/app:v1.0.1');
  assertEquals(tags[1], 'ghcr.io/interledger/builders/app:v1.0');
  assertEquals(tags[2], 'ghcr.io/interledger/builders/app:v1');
})) passed++; else failed++;

if (test('Scenario: Minor release v1.1.0', () => {
  const tags = generateImageTags('interledger/builders/app', 'v1.1.0', 'ghcr.io');
  
  // New v1.1 tag created, v1 updated
  assertEquals(tags[0], 'ghcr.io/interledger/builders/app:v1.1.0');
  assertEquals(tags[1], 'ghcr.io/interledger/builders/app:v1.1');
  assertEquals(tags[2], 'ghcr.io/interledger/builders/app:v1');
})) passed++; else failed++;

if (test('Scenario: Major release v2.0.0', () => {
  const tags = generateImageTags('interledger/builders/app', 'v2.0.0', 'ghcr.io');
  
  // New v2 and v2.0 tags, v1 stays at old version
  assertEquals(tags[0], 'ghcr.io/interledger/builders/app:v2.0.0');
  assertEquals(tags[1], 'ghcr.io/interledger/builders/app:v2.0');
  assertEquals(tags[2], 'ghcr.io/interledger/builders/app:v2');
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
