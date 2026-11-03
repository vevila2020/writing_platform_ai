#!/usr/bin/env node
// Guard for this fork's Go module paths.
//
// Upstream Denova still imports its own module paths, so cherry-picking or
// copying an upstream commit reintroduces them. Those imports compile only by
// accident (Go treats them as a different module) and the version injection in
// `-X denova/internal/buildinfo.Version` fails silently, leaving release
// binaries reporting "dev".
//
// Usage:
//   node scripts/module-paths.mjs           # check, exit 1 on legacy paths
//   node scripts/module-paths.mjs --fix     # rewrite legacy paths in place
//
// The rewrites are deliberately narrow: only import specs and the version
// injection point. String literals such as ".inkdock/runs", "refs/denova/current"
// and "denova/lore" are data, not module paths, and must stay untouched.

import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

export const ROOT_MODULE = 'github.com/vevila2020/writing_platform_ai';
export const AGENT_MODULE = `${ROOT_MODULE}/agent`;

const LEGACY_AGENT_MODULE = 'github.com/alfredxw/denova/agent';
const LEGACY_ROOT_MODULE = 'denova';

// The guard necessarily quotes the legacy paths as patterns and documentation,
// so it skips its own sources instead of rewriting or reporting them.
const SELF_FILES = new Set(['scripts/module-paths.mjs', 'scripts/module-paths.test.mjs']);

// Go sources plus the manifests, docs, and build scripts that quote module paths.
const SCANNED_EXTENSIONS = ['.go', '.mod', '.md', '.sh', '.ps1', '.mjs', '.toml', '.json', '.yml', '.yaml'];

// An import spec for the root module: optional `import` keyword, optional alias,
// then a quoted spec starting with "denova/". A line that merely contains a
// string literal like ".inkdock/runs" or a fixture name like "denova/link" does
// not match, because the opening brace or another token sits before the quote.
const LEGACY_ROOT_IMPORT = /^[ \t]*(?:import[ \t]+)?(?:[A-Za-z_][A-Za-z0-9_]*[ \t]+)?"denova\//;
const LEGACY_ROOT_IMPORT_REWRITE = /(^[ \t]*(?:import[ \t]+)?(?:[A-Za-z_][A-Za-z0-9_]*[ \t]+)?")(denova\/)/;
const LEGACY_VERSION_INJECTION = /(^|[^A-Za-z0-9_-])(-X[ \t]+)denova\//;
const LEGACY_VERSION_INJECTION_REWRITE = /(-X[ \t]+)denova\//;

// A module directive this fork owns, in either manifest.
const LEGACY_MODULE_DIRECTIVE = /^module[ \t]+(?:denova|github\.com\/alfredxw\/denova(?:\/agent)?)[ \t]*$/;

// Reports every legacy module path in one file's content. `file` selects the
// expected module for a `module` directive; pass an empty string when the
// caller only wants import, injection, and path checks.
export function modulePathViolations(file, content) {
  if (SELF_FILES.has(file)) {
    return [];
  }
  const violations = [];
  const lines = content.split(/\r?\n/);
  for (const [index, line] of lines.entries()) {
    const location = `${file}:${index + 1}`;
    // One line carries one kind of path; report the most specific match so a
    // single line does not produce two messages.
    if (LEGACY_MODULE_DIRECTIVE.test(line)) {
      violations.push({ location, message: `legacy module directive (${line.trim()})` });
    } else if (line.includes(LEGACY_AGENT_MODULE)) {
      violations.push({ location, message: `legacy agent module path (${LEGACY_AGENT_MODULE})` });
    } else if (LEGACY_ROOT_IMPORT.test(line)) {
      violations.push({ location, message: `legacy root module import ("${LEGACY_ROOT_MODULE}/...")` });
    } else if (LEGACY_VERSION_INJECTION.test(line)) {
      violations.push({ location, message: `legacy version injection (-X ${LEGACY_ROOT_MODULE}/internal/buildinfo.Version)` });
    }
  }
  return violations;
}

// Rewrites the legacy paths this fork owns. `file` decides which module a
// directive-less manifest belongs to. Content that has no legacy path is
// returned unchanged so callers can skip the write.
export function rewriteModulePaths(content, file = '') {
  let result = content;
  if (result.includes(LEGACY_AGENT_MODULE)) {
    result = result.split(LEGACY_AGENT_MODULE).join(AGENT_MODULE);
  }
  result = result.replace(
    new RegExp(LEGACY_ROOT_IMPORT_REWRITE.source, 'gm'),
    `$1${ROOT_MODULE}/`,
  );
  result = result.replace(
    new RegExp(LEGACY_VERSION_INJECTION_REWRITE.source, 'gm'),
    `$1${ROOT_MODULE}/`,
  );
  if (file.split(/[\\/]/).join('/').endsWith('go.mod')) {
    const expected = file.split(/[\\/]/).join('/') === 'agent/go.mod' ? AGENT_MODULE : ROOT_MODULE;
    result = result.replace(
      new RegExp(LEGACY_MODULE_DIRECTIVE.source, 'gm'),
      `module ${expected}`,
    );
  }
  return result;
}

// The module directives are the identity the guard protects, so assert them
// rather than reading an expected value out of the files being checked.
function moduleDirectiveViolations(root) {
  const violations = [];
  for (const [relative, expected] of [
    ['go.mod', ROOT_MODULE],
    [join('agent', 'go.mod'), AGENT_MODULE],
  ]) {
    let content;
    try {
      content = readFileSync(join(root, relative), 'utf8');
    } catch {
      violations.push({ location: relative, message: 'module manifest is missing' });
      continue;
    }
    const actual = content.split(/\r?\n/).find((line) => line.startsWith('module '));
    if (actual?.trim() !== `module ${expected}`) {
      violations.push({
        location: relative,
        message: `module directive is ${JSON.stringify(actual?.trim() ?? '')}, expected "module ${expected}"`,
      });
    }
  }
  return violations;
}

// Lists tracked files plus untracked files that are not gitignored, so a
// freshly created source file is checked before the first `git add`.
export function listRepositoryFiles(root) {
  const candidates = execFileSync(
    'git',
    ['ls-files', '-z', '--cached', '--others', '--exclude-standard'],
    { cwd: root, encoding: 'utf8' },
  )
    .split('\0')
    .filter(Boolean);
  return [...new Set(candidates)].filter((file) =>
    SCANNED_EXTENSIONS.some((extension) => file.endsWith(extension)),
  );
}

export function checkRepository(root) {
  const violations = moduleDirectiveViolations(root);
  for (const file of listRepositoryFiles(root)) {
    let content;
    try {
      content = readFileSync(join(root, file), 'utf8');
    } catch {
      continue;
    }
    violations.push(...modulePathViolations(file.split(/[\\/]/).join('/'), content));
  }
  return violations;
}

export function fixRepository(root) {
  const rewritten = [];
  for (const file of listRepositoryFiles(root)) {
    if (SELF_FILES.has(file.split(/[\\/]/).join('/'))) {
      continue;
    }
    const path = join(root, file);
    let content;
    try {
      content = readFileSync(path, 'utf8');
    } catch {
      continue;
    }
    const next = rewriteModulePaths(content, file);
    if (next !== content) {
      writeFileSync(path, next, 'utf8');
      rewritten.push(file);
    }
  }
  return rewritten;
}

function main(argv) {
  const root = fileURLToPath(new URL('..', import.meta.url));
  const fix = argv.includes('--fix');
  const unknown = argv.filter((argument) => argument !== '--fix');
  if (unknown.length > 0) {
    console.error(`Unknown argument: ${unknown.join(' ')}`);
    console.error('Usage: node scripts/module-paths.mjs [--fix]');
    return 2;
  }

  if (fix) {
    const rewritten = fixRepository(root);
    if (rewritten.length === 0) {
      console.log('Module paths already use the fork module.');
      return 0;
    }
    for (const file of rewritten) {
      console.log(`Rewrote ${file}`);
    }
    const remaining = checkRepository(root);
    if (remaining.length > 0) {
      console.error('Legacy module paths remain after --fix:');
      for (const { location, message } of remaining) {
        console.error(`  ${location}: ${message}`);
      }
      return 1;
    }
    console.log(`Rewrote ${rewritten.length} file(s).`);
    return 0;
  }

  const violations = checkRepository(root);
  if (violations.length === 0) {
    console.log(`Module paths use ${ROOT_MODULE} and ${AGENT_MODULE}.`);
    return 0;
  }
  console.error('Legacy InkDock module paths found. Run `node scripts/module-paths.mjs --fix`.');
  for (const { location, message } of violations) {
    console.error(`  ${location}: ${message}`);
  }
  console.error(`${violations.length} violation(s).`);
  return 1;
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  process.exit(main(process.argv.slice(2)));
}