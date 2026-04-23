#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const repoRoot = path.resolve(__dirname, "..", "..");
const changelogPath = path.join(repoRoot, "CHANGELOG.md");

function getArg(name) {
  const idx = process.argv.indexOf(name);
  if (idx === -1) return null;
  return process.argv[idx + 1] || null;
}

function hasFlag(name) {
  return process.argv.includes(name);
}

function runGit(args) {
  return execFileSync("git", args, {
    cwd: repoRoot,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  }).trim();
}

function getPreviousTag() {
  try {
    const tags = runGit(["tag", "--sort=-creatordate"]).split("\n").filter(Boolean);
    return tags[0] || null;
  } catch {
    return null;
  }
}

function getLogRange(previousTag) {
  return previousTag ? `${previousTag}..HEAD` : "HEAD";
}

function parseCommits(range) {
  const output = runGit(["log", "--pretty=format:%s", range]);
  const lines = output.split("\n").map((line) => line.trim()).filter(Boolean);
  const sections = {
    Added: [],
    Changed: [],
    Fixed: [],
    Documentation: [],
    CI: [],
    Chore: [],
  };

  const mapping = {
    feat: "Added",
    fix: "Fixed",
    perf: "Fixed",
    refactor: "Changed",
    build: "Changed",
    docs: "Documentation",
    ci: "CI",
    chore: "Chore",
    test: "Chore",
    revert: "Chore",
  };

  for (const subject of lines) {
    const match = subject.match(/^([a-z]+)(\([^)]+\))?:\s+(.*)$/i);
    if (!match) {
      sections.Changed.push(subject);
      continue;
    }

    const [, type, , description] = match;
    const bucket = mapping[type.toLowerCase()] || "Changed";
    sections[bucket].push(description);
  }

  return sections;
}

function formatSection(title, items) {
  if (!items.length) return "";
  const lines = [`### ${title}`, ""];
  for (const item of items) {
    lines.push(`- ${item}`);
  }
  lines.push("");
  return lines.join("\n");
}

function main() {
  const version = getArg("--version");
  const date = getArg("--date") || new Date().toISOString().slice(0, 10);
  const dryRun = hasFlag("--dry-run");

  if (!version) {
    console.error("Usage: node scripts/release/update-changelog.cjs --version vX.Y.Z [--date YYYY-MM-DD] [--dry-run]");
    process.exit(1);
  }

  const previousTag = getPreviousTag();
  const range = getLogRange(previousTag);
  const sections = parseCommits(range);

  const header = fs.existsSync(changelogPath)
    ? fs.readFileSync(changelogPath, "utf8")
    : "# Changelog\n\nAll notable changes to `go-mirofish` will be documented in this file.\n";

  const releaseLines = [
    `## ${version} - ${date}`,
    "",
    previousTag
      ? `Changes since \`${previousTag}\`.`
      : "Initial release notes generated from repository history.",
    "",
    formatSection("Added", sections.Added).trimEnd(),
    formatSection("Changed", sections.Changed).trimEnd(),
    formatSection("Fixed", sections.Fixed).trimEnd(),
    formatSection("Documentation", sections.Documentation).trimEnd(),
    formatSection("CI", sections.CI).trimEnd(),
    formatSection("Chore", sections.Chore).trimEnd(),
  ].filter(Boolean);

  const insertion = `${releaseLines.join("\n\n")}\n\n`;

  if (header.includes(`## ${version} - ${date}`) || header.includes(`## ${version}\n`)) {
    console.error(`CHANGELOG already contains a section for ${version}`);
    process.exit(1);
  }

  const marker = /^# Changelog[^\n]*\n(?:\n[^\n].*?\n(?:[^\n].*\n)*)?/m;
  const match = header.match(marker);

  let next;
  if (match) {
    const insertAt = match.index + match[0].length;
    next = `${header.slice(0, insertAt)}\n${insertion}${header.slice(insertAt).replace(/^\n+/, "")}`;
  } else {
    next = `# Changelog\n\n${insertion}${header}`;
  }

  if (dryRun) {
    console.log(insertion.trimEnd());
    return;
  }

  fs.writeFileSync(changelogPath, next);
  console.log(`Updated ${path.relative(repoRoot, changelogPath)} for ${version}`);
}

main();
