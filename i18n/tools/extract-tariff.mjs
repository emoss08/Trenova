// extract-tariff.mjs collects the doc comments of the Tariff standard library
// (services/crates/rules/src/tariff/stdlib/*.tariff, docs/engineering/formula-language.md).
//
// The rules crate sends a standard library doc to the hosts as a message, not as text: its
// template is the `///` block with the slashes removed and its whitespace collapsed (lines
// joined with single spaces, trimmed), which is also the form the catalogs key it by
// (crate::tariff::stdlib::docs). A function's doc also gives each parameter a doc of its own:
// the sentences of the collapsed doc (each ending at ". ") that name the parameter in
// backticks, joined with a space (crate::tariff::docs::param_text). This module computes the
// same templates from the same sources, so the two must change together; the crate's test
// `every_template_is_in_the_catalog_for_both_hosts` fails when they disagree.
//
// Only a file whose header is `library "std.…"` is read: the doc comments of tariffs and
// organisation libraries are their authors' words and are never translated.

/** A failure the extraction must not get past: the message says where and why. */
export class TariffExtractError extends Error {
  constructor(file, line, reason) {
    super(`${file}:${line}: ${reason}`);
    this.name = "TariffExtractError";
  }
}

const HEADER = /^\s*library\s+"(std\.[^"]*)"/;
const FN = /^\s*(private\s+)?fn\s/;
const NAME = /^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:/;

/** collapse joins a doc's lines with single spaces and trims it, as the crate does. */
export function collapse(text) {
  return text.split(/\s+/).filter((word) => word !== "").join(" ");
}

function isDoc(line) {
  const text = line.trimStart();
  return text.startsWith("///") && !text.startsWith("////");
}

/**
 * paramText is a parameter's doc: the sentences of a function's doc that name it in
 * backticks, joined with a space, or null when none does.
 */
export function paramText(doc, param) {
  const needle = `\`${param}\``;
  let rest = collapse(doc);
  const sentences = [];
  for (let end = rest.indexOf(". "); end !== -1; end = rest.indexOf(". ")) {
    sentences.push(rest.slice(0, end + 2));
    rest = rest.slice(end + 2);
  }
  if (rest !== "") sentences.push(rest);
  const naming = sentences.map((sentence) => sentence.trim()).filter((s) => s.includes(needle));
  return naming.length === 0 ? null : naming.join(" ");
}

const OPENERS = { "(": ")", "[": "]", "{": "}" };

/**
 * fnParams returns the parameter names of the function declared at `text`'s start
 * (`fn name(a: T, b: U = default) -> R`), or [] when its parameter list is not closed.
 */
export function fnParams(text) {
  const open = text.indexOf("(");
  if (open === -1) return [];
  const segments = [];
  const stack = [];
  let segment = "";
  for (let i = open + 1; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === '"') {
      const end = closingQuote(text, i);
      if (end === -1) return [];
      segment += text.slice(i, end + 1);
      i = end;
      continue;
    }
    if (ch in OPENERS) stack.push(OPENERS[ch]);
    else if (ch === stack.at(-1)) stack.pop();
    else if (ch === ")" && stack.length === 0) {
      segments.push(segment);
      return segments.map((s) => NAME.exec(s)?.[1]).filter((name) => name !== undefined);
    } else if (ch === "," && stack.length === 0) {
      segments.push(segment);
      segment = "";
      continue;
    }
    segment += ch;
  }
  return [];
}

function closingQuote(text, start) {
  for (let i = start + 1; i < text.length; i += 1) {
    if (text[i] === "\\") i += 1;
    else if (text[i] === '"') return i;
    else if (text[i] === "\n") return -1;
  }
  return -1;
}

/**
 * extractTariffDocs returns the templates a standard library source's doc comments give
 * ({ message, line }, line where the doc starts; each template once, where it first appears),
 * or [] for any other Tariff file. Throws
 * TariffExtractError on a doc no catalog could hold.
 */
export function extractTariffDocs(source, file = "<source>") {
  const lines = source.replace(/\r\n/g, "\n").split("\n");
  const first = lines.find((line) => line.trim() !== "");
  if (first === undefined || !HEADER.test(first)) return [];

  const out = [];
  const seen = new Set();
  const add = (message, line) => {
    if (/[{}]/.test(message)) {
      throw new TariffExtractError(
        file,
        line,
        "a standard library doc comment has no `{` or `}`: the hosts would read it as an ICU placeholder",
      );
    }
    if (seen.has(message)) return;
    seen.add(message);
    out.push({ message, line });
  };

  let i = 0;
  while (i < lines.length) {
    if (!isDoc(lines[i])) {
      i += 1;
      continue;
    }
    const start = i + 1;
    const parts = [];
    while (i < lines.length && isDoc(lines[i])) {
      parts.push(lines[i].trimStart().slice(3));
      i += 1;
    }
    const doc = collapse(parts.join(" "));
    if (doc === "") continue;
    add(doc, start);
    if (i < lines.length && FN.test(lines[i])) {
      for (const param of fnParams(lines.slice(i).join("\n"))) {
        const text = paramText(doc, param);
        if (text !== null) add(text, start);
      }
    }
  }
  return out;
}
