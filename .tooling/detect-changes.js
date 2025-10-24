#!/usr/bin/env node

/**
 * Detects which builder folders have changes between commits.
 * This script is designed to work in GitHub Actions context.
 * 
 * Outputs a JSON array of folder names to stdout.
 */

import { readdir, access } from 'fs/promises';
import { join } from 'path';
import { execSync } from 'child_process';
import { constants } from 'fs';

/**
 * Check if a path exists
 * @param {string} path 
 * @returns {Promise<boolean>}
 */
async function exists(path) {
  try {
    await access(path, constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

/**
 * Get all folders at root level that contain a Dockerfile.
 * Excludes hidden folders and the .tooling folder.
 * @param {string} rootDir 
 * @returns {Promise<string[]>}
 */
async function getBuildableFolders(rootDir) {
  const entries = await readdir(rootDir, { withFileTypes: true });
  const folders = [];
  
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    if (entry.name.startsWith('.')) continue;
    if (entry.name === '.tooling') continue;
    
    const dockerfilePath = join(rootDir, entry.name, 'Dockerfile');
    if (await exists(dockerfilePath)) {
      folders.push(entry.name);
    }
  }
  
  return folders.sort();
}

/**
 * Execute a git command and return stdout
 * @param {string} command 
 * @returns {string}
 */
function runGit(command) {
  try {
    return execSync(command, { encoding: 'utf8' }).trim();
  } catch (error) {
    console.error(`Git command failed: ${command}`);
    console.error(error.message);
    return '';
  }
}

/**
 * Get list of changed files between two commits
 * @param {string} baseRef 
 * @param {string} headRef 
 * @returns {string[]}
 */
function getChangedFiles(baseRef, headRef = 'HEAD') {
  const output = runGit(`git diff --name-only ${baseRef} ${headRef}`);
  return output ? output.split('\n').filter(Boolean) : [];
}

/**
 * Filter folders that have changes
 * @param {string[]} allFolders 
 * @param {string[]} changedFiles 
 * @returns {string[]}
 */
function filterChangedFolders(allFolders, changedFiles) {
  return allFolders.filter(folder => {
    return changedFiles.some(file => file.startsWith(`${folder}/`));
  });
}

/**
 * Find the git repository root
 * @returns {string}
 */
function getRepoRoot() {
  try {
    return execSync('git rev-parse --show-toplevel', { encoding: 'utf8' }).trim();
  } catch (error) {
    console.error('Failed to find git repository root');
    throw error;
  }
}

/**
 * Main execution
 */
async function main() {
  const args = process.argv.slice(2);
  const mode = args[0] || 'detect';
  
  // Always use the git repository root, not the current working directory
  // This ensures the script works correctly even when run from subdirectories
  const rootDir = getRepoRoot();
  
  // Get all buildable folders
  const allFolders = await getBuildableFolders(rootDir);
  
  if (mode === 'list') {
    // Just list all buildable folders
    console.log(JSON.stringify(allFolders));
    return;
  }
  
  // Detect mode: find which folders have changes
  const eventName = process.env.GITHUB_EVENT_NAME || '';
  const baseRef = process.env.BASE_REF || '';
  
  let changedFolders = [];
  
  if (eventName === 'workflow_dispatch' || !baseRef) {
    // Build all folders on manual dispatch or if we can't determine base
    console.error('Building all folders (workflow_dispatch or no base ref)');
    changedFolders = allFolders;
  } else {
    // Get changed files
    const changedFiles = getChangedFiles(baseRef);
    console.error(`Found ${changedFiles.length} changed files`);
    
    if (changedFiles.length === 0) {
      console.error('No changes detected, building all folders');
      changedFolders = allFolders;
    } else {
      // Filter to folders with changes
      changedFolders = filterChangedFolders(allFolders, changedFiles);
      console.error(`Folders with changes: ${changedFolders.join(', ')}`);
    }
  }
  
  // Output as JSON array
  console.log(JSON.stringify(changedFolders));
}

main().catch(error => {
  console.error('Error:', error.message);
  process.exit(1);
});
