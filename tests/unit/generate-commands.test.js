/**
 * Unit Tests for generate-commands.js
 *
 * Tests the YAML-to-markdown command generator logic.
 * Uses inline spec objects (not actual YAML files) to test
 * the generateForProvider function in isolation.
 *
 * @module tests/unit/generate-commands.test
 */

const test = require('ava');

// The generator module exports generateForProvider for unit testing.
// This will fail until the generator is created (TDD RED phase).
const { generateForProvider } = require('../../bin/generate-commands.js');

// --- Test 1: Shared description frontmatter ---

test('generates correct frontmatter with shared description', t => {
  const spec = { name: 'ant:test', description: 'Test command', body: 'Hello world' };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('---\nname: ant:test\ndescription: "Test command"\n---\n'));
});

test('generates correct frontmatter for opencode with shared description', t => {
  const spec = { name: 'ant:test', description: 'Test command', body: 'Hello world' };
  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('---\nname: ant:test\ndescription: "Test command"\n---\n'));
});

// --- Test 2: Provider-specific description ---

test('uses description_opencode for opencode provider', t => {
  const spec = {
    name: 'ant:test',
    description: 'Plain description',
    description_opencode: 'Emoji Plain description',
    body: 'Hello'
  };
  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('description: "Emoji Plain description"'));
});

test('uses shared description for claude when description_opencode exists', t => {
  const spec = {
    name: 'ant:test',
    description: 'Plain description',
    description_opencode: 'Emoji Plain description',
    body: 'Hello'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('description: "Plain description"'));
});

test('uses description_claude for claude when present', t => {
  const spec = {
    name: 'ant:test',
    description: 'Shared',
    description_claude: 'Claude specific',
    body: 'Hello'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('description: "Claude specific"'));
});

// --- Test 3: ARGUMENTS replacement ---

test('replaces {{ARGUMENTS}} with $ARGUMENTS for claude', t => {
  const spec = { name: 'ant:test', description: 'Test', body: 'The arg is: {{ARGUMENTS}}' };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('The arg is: $ARGUMENTS'));
  t.false(claude.includes('{{ARGUMENTS}}'));
});

test('replaces {{ARGUMENTS}} with $normalized_args for opencode', t => {
  const spec = { name: 'ant:test', description: 'Test', body: 'The arg is: {{ARGUMENTS}}' };
  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('The arg is: $normalized_args'));
  t.false(opencode.includes('{{ARGUMENTS}}'));
});

test('replaces multiple {{ARGUMENTS}} occurrences', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: 'First: {{ARGUMENTS}} and second: {{ARGUMENTS}}'
  };
  const claude = generateForProvider(spec, 'claude');
  t.is(claude.split('$ARGUMENTS').length - 1, 2);
});

// --- Test 4: TOOL_PREFIX replacement ---

test('replaces {{TOOL_PREFIX "desc"}} with Bash tool phrasing for claude', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: '{{TOOL_PREFIX "Setting colony focus..."}}\n```bash\necho hello\n```'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('Run using the Bash tool with description "Setting colony focus...":'));
  t.false(claude.includes('{{TOOL_PREFIX'));
});

test('replaces {{TOOL_PREFIX "desc"}} with Run: for opencode', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: '{{TOOL_PREFIX "Setting colony focus..."}}\n```bash\necho hello\n```'
  };
  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('Run:'));
  t.false(opencode.includes('Run using the Bash tool'));
  t.false(opencode.includes('{{TOOL_PREFIX'));
});

// --- Test 5: Provider-exclusive blocks ---

test('keeps {{#claude}} content for claude, strips for opencode', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: 'Shared content\n{{#claude}}\nClaude only section\n{{/claude}}\nMore shared'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('Claude only section'));
  t.true(claude.includes('Shared content'));
  t.true(claude.includes('More shared'));
  t.false(claude.includes('{{#claude}}'));
  t.false(claude.includes('{{/claude}}'));

  const opencode = generateForProvider(spec, 'opencode');
  t.false(opencode.includes('Claude only section'));
  t.true(opencode.includes('Shared content'));
  t.true(opencode.includes('More shared'));
});

test('keeps {{#opencode}} content for opencode, strips for claude', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: 'Shared content\n{{#opencode}}\nOpenCode only section\n{{/opencode}}\nMore shared'
  };
  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('OpenCode only section'));
  t.true(opencode.includes('Shared content'));
  t.false(opencode.includes('{{#opencode}}'));
  t.false(opencode.includes('{{/opencode}}'));

  const claude = generateForProvider(spec, 'claude');
  t.false(claude.includes('OpenCode only section'));
  t.true(claude.includes('Shared content'));
});

// --- Test 6: Both provider blocks in same body ---

test('handles both provider blocks in same body', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: 'Start\n{{#claude}}\nClaude part\n{{/claude}}\nMiddle\n{{#opencode}}\nOpenCode part\n{{/opencode}}\nEnd'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('Claude part'));
  t.false(claude.includes('OpenCode part'));
  t.true(claude.includes('Start'));
  t.true(claude.includes('Middle'));
  t.true(claude.includes('End'));

  const opencode = generateForProvider(spec, 'opencode');
  t.false(opencode.includes('Claude part'));
  t.true(opencode.includes('OpenCode part'));
  t.true(opencode.includes('Start'));
  t.true(opencode.includes('Middle'));
  t.true(opencode.includes('End'));
});

// --- Test 7: Normalize-args preamble ---

test('injects normalize-args preamble for opencode only', t => {
  const spec = { name: 'ant:test', description: 'Test', body: 'Body content here' };

  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('### Step -1: Normalize Arguments'));
  t.true(opencode.includes('normalized_args=$(bash .aether/aether-utils.sh normalize-args'));

  const claude = generateForProvider(spec, 'claude');
  t.false(claude.includes('### Step -1: Normalize Arguments'));
  t.false(claude.includes('normalize-args'));
});

test('preamble appears after frontmatter before body content', t => {
  const spec = { name: 'ant:test', description: 'Test', body: 'Body content here' };
  const opencode = generateForProvider(spec, 'opencode');

  const frontmatterEnd = opencode.indexOf('---\n\n', 4);
  const preambleStart = opencode.indexOf('### Step -1');
  const bodyStart = opencode.indexOf('Body content here');

  t.true(frontmatterEnd < preambleStart);
  t.true(preambleStart < bodyStart);
});

// --- Test 8: body_claude / body_opencode ---

test('uses body_claude for claude when present', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body_claude: 'Claude-specific body content',
    body_opencode: 'OpenCode-specific body content'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('Claude-specific body content'));
  t.false(claude.includes('OpenCode-specific body content'));
});

test('uses body_opencode for opencode when present', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body_claude: 'Claude-specific body content',
    body_opencode: 'OpenCode-specific body content'
  };
  const opencode = generateForProvider(spec, 'opencode');
  t.true(opencode.includes('OpenCode-specific body content'));
  t.false(opencode.includes('Claude-specific body content'));
});

test('body_claude/body_opencode skip template processing', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body_claude: 'Content with {{ARGUMENTS}} literal',
    body_opencode: 'Content with {{ARGUMENTS}} literal'
  };
  // When using provider-specific bodies, template markers are NOT processed
  // because the content is already provider-specific
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('{{ARGUMENTS}}'));
});

test('falls back to body field when no provider-specific body', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body: 'Shared body with {{ARGUMENTS}}'
  };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('$ARGUMENTS'));
});

// --- Test 9: Error on missing body ---

test('throws when no body field and no provider-specific body', t => {
  const spec = { name: 'ant:test', description: 'Test' };
  const error = t.throws(() => generateForProvider(spec, 'claude'));
  t.true(error.message.includes('ant:test'));
});

test('does not throw when body_claude exists but no body', t => {
  const spec = {
    name: 'ant:test',
    description: 'Test',
    body_claude: 'Claude body',
    body_opencode: 'OpenCode body'
  };
  t.notThrows(() => generateForProvider(spec, 'claude'));
});

// --- Test 10: Generated file header comment ---

test('output includes generated header comment', t => {
  const spec = { name: 'ant:test', description: 'Test', body: 'Hello' };
  const claude = generateForProvider(spec, 'claude');
  t.true(claude.includes('<!-- Generated from .aether/commands/'));
  t.true(claude.includes('DO NOT EDIT DIRECTLY'));
});
