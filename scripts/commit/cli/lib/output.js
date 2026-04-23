/**
 * CLI output formatting. Respects NO_COLOR.
 * Uses unicode symbols per cli-building skill (no emojis).
 */

const noColor = !!process.env.NO_COLOR;

const c = noColor
  ? { reset: '', bright: '', red: '', green: '', yellow: '', blue: '', cyan: '', dim: '' }
  : {
      reset: '\x1b[0m',
      bright: '\x1b[1m',
      red: '\x1b[31m',
      green: '\x1b[32m',
      yellow: '\x1b[33m',
      blue: '\x1b[34m',
      cyan: '\x1b[36m',
      dim: '\x1b[2m',
    };

const symbols = { ok: '✓', fail: '✗', warn: '⚠', arrow: '→' };

function style(msg, color = 'reset') {
  return `${c[color]}${msg}${c.reset}`;
}

function ok(msg) {
  return `${c.green}${symbols.ok}${c.reset} ${msg}`;
}

function fail(msg) {
  return `${c.red}${symbols.fail}${c.reset} ${msg}`;
}

function warn(msg) {
  return `${c.yellow}${symbols.warn}${c.reset} ${msg}`;
}

function arrow(msg) {
  return `${c.blue}${symbols.arrow}${c.reset} ${msg}`;
}

module.exports = { style, ok, fail, warn, arrow, c, symbols };
