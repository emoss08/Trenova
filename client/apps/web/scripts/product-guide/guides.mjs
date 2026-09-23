/**
 * Reads the hand-written guides under docs/product-guide.
 *
 * A guide is markdown with a small, strict shape, because it is data an agent
 * reads back to a person, and a loose shape would let a guide say things the
 * checks cannot see:
 *
 *   ---
 *   path: /billing/configuration-files/rate-matrices
 *   aliases: [rate table, lane rates]
 *   related:
 *     - /billing/configuration-files/customers
 *   covers:
 *     - /billing/configuration-files/rate-matrices/new
 *   ---
 *
 *   ## What it's for
 *   One or more paragraphs.
 *
 *   ## Tasks
 *
 *   ### Add a rate matrix
 *   Keywords: new rate, lane pricing
 *   1. Open **Rate matrices**.
 *   2. Select **New rate matrix**.
 *
 *   ## Notes
 *   Optional paragraphs.
 *
 * Text in **bold** is a label on screen and is checked against the app's own
 * strings; a markdown link must point at a page the catalog knows.
 */

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const FRONTMATTER_KEYS = new Set(["path", "title", "aliases", "related", "covers"]);
const SECTIONS = new Map([
  ["what it's for", "summary"],
  ["tasks", "tasks"],
  ["notes", "notes"],
]);

function unquote(value) {
  const trimmed = value.trim();
  if (
    trimmed.length >= 2 &&
    ((trimmed.startsWith('"') && trimmed.endsWith('"')) ||
      (trimmed.startsWith("'") && trimmed.endsWith("'")))
  ) {
    return trimmed.slice(1, -1);
  }

  return trimmed;
}

function parseFrontmatter(lines, file, errors) {
  const data = {};
  let key = null;
  for (const line of lines) {
    if (line.trim() === "") {
      continue;
    }
    const item = /^\s+-\s+(.*)$/.exec(line);
    if (item) {
      if (key === null || !Array.isArray(data[key])) {
        errors.push(`${file}: list item outside a list: ${line.trim()}`);
        continue;
      }
      data[key].push(unquote(item[1]));
      continue;
    }
    const pair = /^([A-Za-z]+):\s*(.*)$/.exec(line);
    if (!pair) {
      errors.push(`${file}: unreadable frontmatter line: ${line.trim()}`);
      continue;
    }
    key = pair[1];
    if (!FRONTMATTER_KEYS.has(key)) {
      errors.push(`${file}: unknown frontmatter key "${key}"`);
    }
    const value = pair[2].trim();
    if (value === "") {
      data[key] = [];
    } else if (value.startsWith("[") && value.endsWith("]")) {
      data[key] = value
        .slice(1, -1)
        .split(",")
        .map(unquote)
        .filter((entry) => entry !== "");
    } else {
      data[key] = unquote(value);
    }
  }

  return data;
}

function paragraphs(lines) {
  return lines
    .join("\n")
    .split(/\n\s*\n/)
    .map((block) => block.replace(/\s*\n\s*/g, " ").trim())
    .filter((block) => block !== "");
}

function parseTasks(lines, file, errors) {
  const tasks = [];
  let task = null;
  for (const raw of lines) {
    const line = raw.trimEnd();
    if (line.trim() === "") {
      continue;
    }
    const heading = /^###\s+(.+)$/.exec(line);
    if (heading) {
      task = { title: heading[1].trim(), keywords: [], steps: [] };
      tasks.push(task);
      continue;
    }
    if (task === null) {
      errors.push(`${file}: text under Tasks before the first ### task: ${line.trim()}`);
      continue;
    }
    const keywords = /^Keywords:\s*(.+)$/i.exec(line.trim());
    if (keywords) {
      task.keywords.push(
        ...keywords[1]
          .split(",")
          .map((keyword) => keyword.trim())
          .filter((keyword) => keyword !== ""),
      );
      continue;
    }
    const step = /^\d+\.\s+(.+)$/.exec(line.trim());
    if (step) {
      task.steps.push(step[1].trim());
      continue;
    }
    if (/^\s{2,}\S/.test(raw) && task.steps.length > 0) {
      task.steps[task.steps.length - 1] += ` ${line.trim()}`;
      continue;
    }
    errors.push(
      `${file}: task "${task.title}" may hold only a Keywords line and numbered steps: ${line.trim()}`,
    );
  }

  for (const entry of tasks) {
    if (entry.steps.length === 0) {
      errors.push(`${file}: task "${entry.title}" has no steps`);
    }
  }

  return tasks;
}

export function parseGuide(file, text, errors) {
  const lines = text.replace(/\r\n/g, "\n").split("\n");
  if (lines[0]?.trim() !== "---") {
    errors.push(`${file}: missing frontmatter`);
    return null;
  }
  const end = lines.indexOf("---", 1);
  if (end === -1) {
    errors.push(`${file}: unterminated frontmatter`);
    return null;
  }

  const front = parseFrontmatter(lines.slice(1, end), file, errors);
  if (typeof front.path !== "string" || !front.path.startsWith("/")) {
    errors.push(`${file}: frontmatter needs path: /the/page`);
    return null;
  }

  const sections = new Map();
  let current = null;
  for (const line of lines.slice(end + 1)) {
    if (/^#\s/.test(line)) {
      continue;
    }
    const heading = /^##\s+(.+)$/.exec(line);
    if (heading) {
      const name = heading[1].trim().toLowerCase();
      current = SECTIONS.get(name) ?? null;
      if (current === null) {
        errors.push(`${file}: unknown section "## ${heading[1].trim()}"`);
      } else if (sections.has(current)) {
        errors.push(`${file}: section "## ${heading[1].trim()}" appears twice`);
      } else {
        sections.set(current, []);
      }
      continue;
    }
    if (current !== null) {
      sections.get(current).push(line);
    }
  }

  const summary = paragraphs(sections.get("summary") ?? []);
  if (summary.length === 0) {
    errors.push(`${file}: "## What it's for" is missing or empty`);
  }
  const tasks = parseTasks(sections.get("tasks") ?? [], file, errors);
  if (tasks.length === 0) {
    errors.push(`${file}: "## Tasks" needs at least one ### task`);
  }

  const asList = (value) => (Array.isArray(value) ? value : value ? [value] : []);

  return {
    file,
    path: front.path,
    title: typeof front.title === "string" ? front.title : "",
    aliases: asList(front.aliases),
    related: asList(front.related),
    covers: asList(front.covers),
    summary: summary.join("\n\n"),
    tasks,
    notes: paragraphs(sections.get("notes") ?? []).join("\n\n"),
  };
}

function markdownFiles(directory) {
  const files = [];
  for (const entry of readdirSync(directory)) {
    const full = join(directory, entry);
    if (statSync(full).isDirectory()) {
      files.push(...markdownFiles(full));
    } else if (entry.endsWith(".md") && entry !== "README.md") {
      files.push(full);
    }
  }

  return files.sort();
}

export function readGuides(directory, repoRoot, errors) {
  const guides = [];
  for (const file of markdownFiles(directory)) {
    const guide = parseGuide(relative(repoRoot, file), readFileSync(file, "utf8"), errors);
    if (guide) {
      guides.push(guide);
    }
  }

  return guides;
}

/** Every **bold** label and every markdown link target in a guide's text. */
export function referencesIn(guide) {
  const texts = [guide.summary, guide.notes, ...guide.tasks.flatMap((task) => task.steps)];
  const labels = [];
  const links = [];
  for (const text of texts) {
    for (const match of text.matchAll(/\*\*([^*]+)\*\*/g)) {
      labels.push(match[1].trim());
    }
    for (const match of text.matchAll(/\]\(([^)\s]+)\)/g)) {
      links.push(match[1]);
    }
  }

  return { labels, links };
}
