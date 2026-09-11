// extract-templates.mjs collects the strings inside {{ t "..." }} in the built-in
// document templates — the emails, in-app notifications and PDFs the system sends.
//
// These are Go template assets rather than Go or TypeScript source, so neither AST
// extractor can see them. The call syntax is unambiguous enough to match directly:
// `t` is a registered template function, `{{ t "..." }}` cannot be anything else, and
// a quoted Go template string escapes with a backslash exactly like JSON.
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";

const TEMPLATE_ROOTS = [
  "services/tms/internal/core/domain/documenttemplate/starters/assets",
];

// {{ t "message" ...args }} and the whitespace-trimming {{- t "..." -}} forms.
const CALL = /\{\{-?\s*t\s+"((?:[^"\\]|\\.)*)"/g;

function unquote(raw) {
  return raw.replace(/\\(["\\nrt])/g, (_, ch) => {
    switch (ch) {
      case "n":
        return "\n";
      case "r":
        return "\r";
      case "t":
        return "\t";
      default:
        return ch;
    }
  });
}

export async function extractTemplates(repoRoot) {
  const entries = [];

  for (const root of TEMPLATE_ROOTS) {
    let files;
    try {
      files = await readdir(join(repoRoot, root));
    } catch {
      continue;
    }

    for (const name of files) {
      const full = join(repoRoot, root, name);
      const source = await readFile(full, "utf8");
      const relPath = relative(repoRoot, full);

      // The kind is the filename without its channel extension, which is what
      // groups a subject with the body it is sent beside.
      const area = `template/${name.replace(/\.(html|txt|subject|css)$/, "")}`;

      for (const match of source.matchAll(CALL)) {
        const message = unquote(match[1]);
        if (message.trim() === "") continue;

        const line = source.slice(0, match.index).split("\n").length;
        entries.push({ message, file: relPath, line, kind: "template", area, scope: "go" });
      }
    }
  }

  return { entries };
}
