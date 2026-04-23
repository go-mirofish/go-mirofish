#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const repoRoot = path.resolve(__dirname, "..", "..");
const changelogPath = path.join(repoRoot, "CHANGELOG.md");

function getArg(name) {
  const idx = process.argv.indexOf(name);
  if (idx === -1) return null;
  return process.argv[idx + 1] || null;
}

function usage() {
  console.error(
    "Usage: node scripts/release/extract-release-notes.cjs --tag vX.Y.Z [--output file]"
  );
}

function extractSection(changelog, tag) {
  const header = `## ${tag}`;
  const start = changelog.indexOf(header);
  if (start === -1) {
    throw new Error(`No CHANGELOG section found for ${tag}`);
  }

  const afterStart = changelog.slice(start);
  const nextMatch = afterStart.slice(header.length).match(/\n##\s+v[^\n]*/);
  const end =
    nextMatch && typeof nextMatch.index === "number"
      ? start + header.length + nextMatch.index + 1
      : changelog.length;

  return changelog.slice(start, end).trim();
}

function main() {
  const tag = getArg("--tag");
  const output = getArg("--output");

  if (!tag) {
    usage();
    process.exit(1);
  }

  if (!fs.existsSync(changelogPath)) {
    throw new Error(`CHANGELOG not found at ${changelogPath}`);
  }

  const changelog = fs.readFileSync(changelogPath, "utf8");
  const section = extractSection(changelog, tag);

  if (output) {
    fs.writeFileSync(path.resolve(output), `${section}\n`, "utf8");
    console.log(`Wrote release notes for ${tag} to ${output}`);
    return;
  }

  console.log(section);
}

main();
