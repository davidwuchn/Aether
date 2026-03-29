#!/usr/bin/env node

/**
 * YAML Command Generator
 *
 * Reads YAML source files from .aether/commands/ and generates
 * both Claude Code and OpenCode command markdown files.
 *
 * Usage:
 *   node bin/generate-commands.js          # Generate all command files
 *   node bin/generate-commands.js --check  # Check if generated files are up-to-date
 *
 * @module bin/generate-commands
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');

const YAML_DIR = path.join(__dirname, '..', '.aether', 'commands');
const CLAUDE_DIR = path.join(__dirname, '..', '.claude', 'commands', 'ant');
const OPENCODE_DIR = path.join(__dirname, '..', '.opencode', 'commands', 'ant');

const NORMALIZE_PREAMBLE = `### Step -1: Normalize Arguments

Run: \`normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")\`

This ensures arguments work correctly in both Claude Code and OpenCode. Use \`$normalized_args\` throughout this command.

`;

/**
 * Generate a command markdown file for a specific provider.
 *
 * @param {object} spec - Parsed YAML command spec
 * @param {string} spec.name - Command name (e.g., "ant:focus")
 * @param {string} spec.description - Shared description
 * @param {string} [spec.description_claude] - Claude-specific description
 * @param {string} [spec.description_opencode] - OpenCode-specific description
 * @param {string} [spec.body] - Shared command body with template markers
 * @param {string} [spec.body_claude] - Claude-specific body (skips template processing)
 * @param {string} [spec.body_opencode] - OpenCode-specific body (skips template processing)
 * @param {string} provider - Target provider: 'claude' or 'opencode'
 * @returns {string} Generated markdown content
 */
function generateForProvider(spec, provider) {
  // 1. Description: use provider-specific if available
  var desc = spec['description_' + provider] || spec.description;

  // 2. Build frontmatter
  var output = '---\nname: ' + spec.name + '\ndescription: "' + desc + '"\n---\n\n';

  // 3. Header comment
  var yamlFilename = spec.name.replace('ant:', '') + '.yaml';
  output = '<!-- Generated from .aether/commands/' + yamlFilename + ' - DO NOT EDIT DIRECTLY -->\n' + output;

  // 4. Determine body source
  var providerBody = spec['body_' + provider];
  var body;

  if (providerBody) {
    // Provider-specific body -- use directly, no template processing
    body = providerBody;
  } else if (spec.body) {
    // Shared body -- apply template processing
    body = spec.body;

    // Replace {{ARGUMENTS}}
    if (provider === 'claude') {
      body = body.replace(/\{\{ARGUMENTS\}\}/g, '$ARGUMENTS');
    } else {
      body = body.replace(/\{\{ARGUMENTS\}\}/g, '$normalized_args');
    }

    // Replace {{TOOL_PREFIX "..."}}
    if (provider === 'claude') {
      body = body.replace(/\{\{TOOL_PREFIX "(.+?)"\}\}/g, 'Run using the Bash tool with description "$1":');
    } else {
      body = body.replace(/\{\{TOOL_PREFIX "(.+?)"\}\}/g, 'Run:');
    }

    // Strip opposite-provider blocks
    var opposite = provider === 'claude' ? 'opencode' : 'claude';
    var stripRegex = new RegExp('\\{\\{#' + opposite + '\\}\\}[\\s\\S]*?\\{\\{/' + opposite + '\\}\\}', 'g');
    body = body.replace(stripRegex, '');

    // Keep same-provider blocks (unwrap markers)
    var keepRegex = new RegExp('\\{\\{#' + provider + '\\}\\}([\\s\\S]*?)\\{\\{/' + provider + '\\}\\}', 'g');
    body = body.replace(keepRegex, '$1');
  } else {
    throw new Error('Command "' + spec.name + '" has no body field and no body_' + provider + ' field');
  }

  // 5. Inject normalize-args preamble for OpenCode
  if (provider === 'opencode') {
    output += NORMALIZE_PREAMBLE;
  }

  output += body;
  return output;
}

/**
 * Main function: read YAML sources, generate command files for both providers.
 */
function main() {
  var checkMode = process.argv.includes('--check');

  // Ensure YAML source directory exists
  if (!fs.existsSync(YAML_DIR)) {
    console.log('No YAML source directory found at ' + YAML_DIR);
    console.log('Run YAML conversion first to create .aether/commands/*.yaml files.');
    process.exit(0);
  }

  // Read all YAML files
  var yamlFiles = fs.readdirSync(YAML_DIR).filter(function (f) {
    return f.endsWith('.yaml');
  });

  if (yamlFiles.length === 0) {
    console.log('No .yaml files found in ' + YAML_DIR);
    process.exit(0);
  }

  var mismatches = [];
  var generated = 0;

  yamlFiles.forEach(function (filename) {
    var filePath = path.join(YAML_DIR, filename);
    var content = fs.readFileSync(filePath, 'utf8');
    var spec = yaml.load(content);
    var outputName = filename.replace('.yaml', '.md');

    var providers = [
      { name: 'claude', dir: CLAUDE_DIR },
      { name: 'opencode', dir: OPENCODE_DIR }
    ];

    providers.forEach(function (prov) {
      var output = generateForProvider(spec, prov.name);
      var outputPath = path.join(prov.dir, outputName);

      if (checkMode) {
        // Compare with existing file
        if (fs.existsSync(outputPath)) {
          var existing = fs.readFileSync(outputPath, 'utf8');
          if (existing !== output) {
            mismatches.push(outputPath);
          }
        } else {
          mismatches.push(outputPath + ' (missing)');
        }
      } else {
        // Ensure output directory exists
        if (!fs.existsSync(prov.dir)) {
          fs.mkdirSync(prov.dir, { recursive: true });
        }
        fs.writeFileSync(outputPath, output);
        generated++;
      }
    });
  });

  if (checkMode) {
    if (mismatches.length > 0) {
      console.log('Generated files are out of date:');
      mismatches.forEach(function (m) {
        console.log('  ' + m);
      });
      console.log('\nRun "node bin/generate-commands.js" to regenerate.');
      process.exit(1);
    } else {
      console.log('All generated files are up to date. (' + yamlFiles.length + ' YAML sources checked)');
      process.exit(0);
    }
  } else {
    console.log('Generated ' + generated + ' command files from ' + yamlFiles.length + ' YAML sources.');
  }
}

module.exports = { generateForProvider };

if (require.main === module) {
  main();
}
