#!/usr/bin/env node

/**
 * Simple test runner for the tooling scripts.
 * Run with: node test.js
 */

import { execSync } from 'child_process';

function run(command) {
  try {
    const output = execSync(command, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] });
    return { success: true, output: output.trim() };
  } catch (error) {
    return { success: false, error: error.message, output: error.stdout?.toString() || '' };
  }
}

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

console.log('Running tooling tests...\n');

let passed = 0;
let failed = 0;

// Test 1: List all buildable folders
if (test('List all buildable folders', () => {
  const result = run('node detect-changes.js list');
  if (!result.success) throw new Error('Script failed to run');
  
  const folders = JSON.parse(result.output);
  if (!Array.isArray(folders)) throw new Error('Output is not an array');
  
  console.log(`  Found ${folders.length} buildable folder(s): ${folders.join(', ')}`);
})) {
  passed++;
} else {
  failed++;
}

// Test 2: Verify .tooling is excluded
if (test('.tooling folder is excluded from buildable list', () => {
  const result = run('node detect-changes.js list');
  if (!result.success) throw new Error('Script failed to run');
  
  const folders = JSON.parse(result.output);
  if (folders.includes('.tooling')) {
    throw new Error('.tooling should not be in the buildable folders list');
  }
})) {
  passed++;
} else {
  failed++;
}

// Test 3: Verify hidden folders are excluded
if (test('Hidden folders are excluded from buildable list', () => {
  const result = run('node detect-changes.js list');
  if (!result.success) throw new Error('Script failed to run');
  
  const folders = JSON.parse(result.output);
  const hiddenFolders = folders.filter(f => f.startsWith('.'));
  if (hiddenFolders.length > 0) {
    throw new Error(`Hidden folders found: ${hiddenFolders.join(', ')}`);
  }
})) {
  passed++;
} else {
  failed++;
}

// Test 4: Output is valid JSON
if (test('Output is valid JSON', () => {
  const result = run('node detect-changes.js list');
  if (!result.success) throw new Error('Script failed to run');
  
  try {
    JSON.parse(result.output);
  } catch {
    throw new Error('Output is not valid JSON');
  }
})) {
  passed++;
} else {
  failed++;
}

console.log(`\nResults: ${passed} passed, ${failed} failed`);
process.exit(failed > 0 ? 1 : 0);
