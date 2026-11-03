import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  AGENT_MODULE,
  ROOT_MODULE,
  modulePathViolations,
  rewriteModulePaths,
} from './module-paths.mjs';

test('detects the legacy agent module path', () => {
  const content = [
    'import (',
    '\tagentschema "github.com/alfredxw/denova/agent/schema"',
    ')',
  ].join('\n');
  const violations = modulePathViolations('a.go', content);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].location, 'a.go:2');
});

test('detects the legacy root module import', () => {
  for (const line of [
    '\t"denova/internal/app"',
    '\tappsvc "denova/internal/app"',
    'import "denova/config"',
    'import app "denova/internal/app"',
  ]) {
    assert.equal(modulePathViolations('a.go', line).length, 1, line);
  }
});

test('detects the legacy version injection point', () => {
  for (const line of [
    'go build -ldflags "-X denova/internal/buildinfo.Version=${VERSION}" ./cmd/denova',
    'exec.Command("go", "build", "-ldflags", "-X denova/internal/buildinfo.Version=0.2.0")',
  ]) {
    assert.equal(modulePathViolations('build.sh', line).length, 1, line);
  }
});

test('ignores data literals that only look like module paths', () => {
  // Data directories, git refs, tool sources, archive fixture names, the
  // user-agent prefix, and attribution links are not module paths.
  const content = [
    'const runs = ".inkdock/runs"',
    'if entry.Name() == ".inkdock" || entry.Name() == ".denova" {',
    'ref := plumbing.ReferenceName("refs/denova/history")',
    'for _, source := range []ToolSource{"", "denova/lore"} {',
    '{Name: "denova/link", Typeflag: tar.TypeSymlink}',
    'header := &zip.FileHeader{Name: "denova/" + name}',
    'req.Header.Set("User-Agent", "InkDock/"+s.version)',
    'InkDock is based on [upstream](https://github.com/alfredxw/denova).',
  ].join('\n');
  assert.deepEqual(modulePathViolations('a.go', content), []);
});

test('detects legacy module directives', () => {
  assert.equal(modulePathViolations('go.mod', 'module denova\n').length, 1);
  assert.equal(modulePathViolations('agent/go.mod', 'module github.com/alfredxw/denova/agent').length, 1);
  assert.deepEqual(modulePathViolations('go.mod', 'module github.com/vevila2020/writing_platform_ai'), []);
});

test('rewrites a module directive for the manifest it belongs to', () => {
  assert.equal(rewriteModulePaths('module denova\n', 'go.mod'), `module ${ROOT_MODULE}\n`);
  assert.equal(
    rewriteModulePaths('module github.com/alfredxw/denova/agent\n', 'agent/go.mod'),
    `module ${AGENT_MODULE}\n`,
  );
});

test('does not report or rewrite the guard sources themselves', () => {
  const content = "const legacy = 'github.com/alfredxw/denova/agent';\nimport \"denova/x\"";
  for (const file of ['scripts/module-paths.mjs', 'scripts/module-paths.test.mjs']) {
    assert.deepEqual(modulePathViolations(file, content), [], file);
  }
});

test('rewrites import specs without touching data literals', () => {
  const content = [
    'import (',
    '\tagentschema "github.com/alfredxw/denova/agent/schema"',
    '\t"denova/internal/app"',
    ')',
    'const dir = ".inkdock/runs"',
  ].join('\n');
  assert.equal(
    rewriteModulePaths(content, 'a.go'),
    [
      'import (',
      `\tagentschema "${AGENT_MODULE}/schema"`,
      `\t"${ROOT_MODULE}/internal/app"`,
      ')',
      'const dir = ".inkdock/runs"',
    ].join('\n'),
  );
});

test('rewrites the version injection point', () => {
  const content = 'go build -ldflags "-X denova/internal/buildinfo.Version=1.2.3" ./cmd/denova';
  assert.equal(
    rewriteModulePaths(content, 'build.sh'),
    `go build -ldflags "-X ${ROOT_MODULE}/internal/buildinfo.Version=1.2.3" ./cmd/denova`,
  );
});

test('leaves already-correct content unchanged', () => {
  const content = [
    `import "github.com/vevila2020/writing_platform_ai/internal/app"`,
    `import "${AGENT_MODULE}/schema"`,
  ].join('\n');
  assert.equal(rewriteModulePaths(content, 'a.go'), content);
  assert.deepEqual(modulePathViolations('a.go', content), []);
});