// extract-rust.mjs collects the message!("...") templates in the rules crate
// (services/crates/rules, docs/engineering/rules-engine.md).
//
// The crate never translates. Every sentence it returns is an English ICU template written as
// message!("..."), sent over the wire as {"message": template, "args": [...]}, and looked up by
// the browser (translate) and the server (shared/i18n). So each template has to reach both
// catalogs: the Go scope, for the validation errors the server reports, and the web app's
// `web` bundle, for the formula editor and the rule screens, which can open on any page.
//
// The crate is read with a small Rust lexer rather than a regex, so a template is found only
// where the compiler would see one: comments, doc comments (which mention `message!("...")`),
// strings and char literals are skipped, and so is everything under #[cfg(test)], including
// a `#[cfg(test)] mod tests;` file. The argument must be one plain or raw string literal; any
// other argument stops the run, because a template the extractor cannot read would ship
// untranslated with nothing failing.
//
// The template is the catalog key at run time exactly as the crate spells it, while the
// catalogs collapse whitespace in their keys. A template with leading, trailing or repeated
// whitespace, or a newline, would never be found, so it is refused too. A literal wrapped with
// a trailing backslash is fine: Rust drops the newline and the indentation after it.
//
// The crate also ships the Tariff standard library (src/tariff/stdlib/*.tariff), whose doc
// comments it sends as messages too; extract-tariff.mjs reads them, and they are filed like the
// crate's other templates.
import { readdir, readFile } from "node:fs/promises";
import { basename, dirname, join, relative, sep } from "node:path";
import { extractTariffDocs } from "./extract-tariff.mjs";
import { reject } from "./filter.mjs";

export const RUST_ROOTS = ["services/crates/rules/src"];

// The client area a rules template is filed under. Any area under apps/web/ puts a string in
// the web app's startup bundle (bundles.mjs), where every screen that renders it has it.
export const RULES_CLIENT_AREA = "apps/web/rules";

const MACRO = "message";

/** A failure the extraction must not get past: the message says where and why. */
export class RustExtractError extends Error {
  constructor(file, line, reason) {
    super(`${file}:${line}: ${reason}`);
    this.name = "RustExtractError";
  }
}

const IDENT_START = /[A-Za-z_]/;
const IDENT_PART = /[A-Za-z0-9_]/;
const WHITESPACE = /[ \t\n\r]/;

/**
 * tokenize splits Rust source into the tokens extraction needs: identifiers, punctuation,
 * string literals (decoded) and everything else as opaque. Comments vanish.
 */
export function tokenize(source, file = "<source>") {
  const text = source.replace(/\r\n/g, "\n");
  const tokens = [];
  let i = 0;
  let line = 1;

  const advance = (n) => {
    for (let k = 0; k < n; k += 1) if (text[i + k] === "\n") line += 1;
    i += n;
  };
  const fail = (at, reason) => {
    throw new RustExtractError(file, at, reason);
  };

  while (i < text.length) {
    const ch = text[i];
    const next = text[i + 1];

    if (WHITESPACE.test(ch)) {
      advance(1);
      continue;
    }
    if (ch === "/" && next === "/") {
      while (i < text.length && text[i] !== "\n") i += 1;
      continue;
    }
    if (ch === "/" && next === "*") {
      const start = line;
      let depth = 0;
      while (i < text.length) {
        if (text[i] === "/" && text[i + 1] === "*") {
          depth += 1;
          advance(2);
        } else if (text[i] === "*" && text[i + 1] === "/") {
          depth -= 1;
          advance(2);
          if (depth === 0) break;
        } else {
          advance(1);
        }
      }
      if (depth !== 0) fail(start, "unterminated block comment");
      continue;
    }

    const stringStart = stringPrefix(text, i);
    if (stringStart !== null) {
      const start = line;
      const { kind, hashes, prefixLength } = stringStart;
      advance(prefixLength);
      if (hashes === null) {
        let raw = "";
        while (i < text.length && text[i] !== '"') {
          if (text[i] === "\\") {
            raw += text.slice(i, i + 2);
            advance(2);
          } else {
            raw += text[i];
            advance(1);
          }
        }
        if (i >= text.length) fail(start, "unterminated string literal");
        advance(1);
        tokens.push({ type: "string", kind, value: unescape(raw, file, start), line: start });
      } else {
        const close = `"${"#".repeat(hashes)}`;
        const end = text.indexOf(close, i);
        if (end === -1) fail(start, "unterminated raw string literal");
        const value = text.slice(i, end);
        advance(end - i + close.length);
        tokens.push({ type: "string", kind: `raw-${kind}`, value, line: start });
      }
      continue;
    }

    if (ch === "'") {
      const start = line;
      if (next === "\\") {
        const end = text.indexOf("'", i + 2 + (text[i + 2] === "'" ? 1 : 0));
        if (end === -1) fail(start, "unterminated char literal");
        advance(end - i + 1);
        tokens.push({ type: "other", value: "char", line: start });
        continue;
      }
      const charLength = next === undefined ? 1 : String.fromCodePoint(text.codePointAt(i + 1)).length;
      if (text[i + 1 + charLength] === "'") {
        advance(charLength + 2);
        tokens.push({ type: "other", value: "char", line: start });
        continue;
      }
      advance(1);
      while (i < text.length && IDENT_PART.test(text[i])) advance(1);
      tokens.push({ type: "other", value: "lifetime", line: start });
      continue;
    }

    if (ch === "r" && next === "#" && IDENT_START.test(text[i + 2] ?? "")) {
      const start = i + 2;
      advance(2);
      while (i < text.length && IDENT_PART.test(text[i])) advance(1);
      tokens.push({ type: "ident", value: text.slice(start, i), line });
      continue;
    }
    if (IDENT_START.test(ch)) {
      const start = i;
      while (i < text.length && IDENT_PART.test(text[i])) advance(1);
      tokens.push({ type: "ident", value: text.slice(start, i), line });
      continue;
    }
    if (/[0-9]/.test(ch)) {
      while (i < text.length && /[0-9A-Za-z_]/.test(text[i])) advance(1);
      tokens.push({ type: "other", value: "number", line });
      continue;
    }
    tokens.push({ type: "punct", value: ch, line });
    advance(1);
  }
  return tokens;
}

// stringPrefix recognises the start of a string literal at `i`: "...", r"...", r#"..."#,
// and the byte and C forms (b, br, c, cr), which are found so they can be refused.
function stringPrefix(text, i) {
  const match = /^(b|c)?(r(#*))?"/.exec(text.slice(i, i + 260));
  if (match === null) return null;
  if (i > 0 && IDENT_PART.test(text[i - 1])) return null;
  const kind = match[1] === "b" ? "byte" : match[1] === "c" ? "c" : "str";
  return {
    kind,
    hashes: match[2] === undefined ? null : match[3].length,
    prefixLength: match[0].length,
  };
}

/** unescape decodes the body of a non-raw Rust string literal as the compiler would. */
export function unescape(raw, file = "<source>", line = 0) {
  let out = "";
  for (let i = 0; i < raw.length; i += 1) {
    const ch = raw[i];
    if (ch !== "\\") {
      out += ch;
      continue;
    }
    const esc = raw[i + 1];
    i += 1;
    switch (esc) {
      case "n":
        out += "\n";
        break;
      case "r":
        out += "\r";
        break;
      case "t":
        out += "\t";
        break;
      case "\\":
      case "'":
      case '"':
        out += esc;
        break;
      case "0":
        out += "\0";
        break;
      case "x": {
        const hex = raw.slice(i + 1, i + 3);
        if (!/^[0-7][0-9A-Fa-f]$/.test(hex)) {
          throw new RustExtractError(file, line, `invalid \\x escape \\x${hex}`);
        }
        out += String.fromCharCode(Number.parseInt(hex, 16));
        i += 2;
        break;
      }
      case "u": {
        const match = /^\{([0-9A-Fa-f_]{1,8})\}/.exec(raw.slice(i + 1));
        if (match === null) throw new RustExtractError(file, line, "invalid \\u escape");
        out += String.fromCodePoint(Number.parseInt(match[1].replaceAll("_", ""), 16));
        i += match[0].length;
        break;
      }
      case "\n":
        while (i + 1 < raw.length && WHITESPACE.test(raw[i + 1])) i += 1;
        break;
      default:
        throw new RustExtractError(file, line, `unknown escape \\${esc ?? ""}`);
    }
  }
  return out;
}

const OPEN = { "(": ")", "[": "]", "{": "}" };

function closing(tokens, openIndex) {
  const stack = [];
  for (let k = openIndex; k < tokens.length; k += 1) {
    const token = tokens[k];
    if (token.type !== "punct") continue;
    if (token.value in OPEN) stack.push(OPEN[token.value]);
    else if (token.value === stack.at(-1)) {
      stack.pop();
      if (stack.length === 0) return k;
    }
  }
  return -1;
}

// isTestCfg reports whether attribute tokens (between `[` and `]`) are a cfg that holds only
// under test: #[cfg(test)], #[cfg(all(test, ...))]. `not(test)` and `any(test, ...)` also
// hold outside tests, so code under them is extracted.
function isTestCfg(inner) {
  if (inner[0]?.type !== "ident" || inner[0].value !== "cfg") return false;
  const idents = inner.filter((t) => t.type === "ident").map((t) => t.value);
  return idents.includes("test") && !idents.includes("not") && !idents.includes("any");
}

// skipItem returns the index after the item starting at `start`, and, when the item is
// `mod name;` (a module whose body is another file), its name and any #[path = "..."].
function skipItem(tokens, start) {
  let k = start;
  let path = null;
  while (tokens[k]?.value === "#" && tokens[k + 1]?.value === "[") {
    const end = closing(tokens, k + 1);
    if (end === -1) return { next: tokens.length, module: null };
    const [name, equals, value] = tokens.slice(k + 2, end);
    if (name?.value === "path" && equals?.value === "=" && value?.type === "string") {
      path = value.value;
    }
    k = end + 1;
  }
  const first = k;
  let depth = 0;
  for (; k < tokens.length; k += 1) {
    const token = tokens[k];
    if (token.type !== "punct") continue;
    if (token.value === "(" || token.value === "[") depth += 1;
    else if (token.value === ")" || token.value === "]") depth -= 1;
    else if (token.value === ";" && depth === 0) {
      const words = tokens.slice(first, k).filter((t) => t.type === "ident").map((t) => t.value);
      const at = words.lastIndexOf("mod");
      const module = at !== -1 && at === words.length - 2 ? { name: words.at(-1), path } : null;
      return { next: k + 1, module };
    } else if (token.value === "{" && depth === 0) {
      const end = closing(tokens, k);
      if (end === -1) return { next: tokens.length, module: null };
      return { next: tokens[end + 1]?.value === ";" ? end + 2 : end + 1, module: null };
    }
  }
  return { next: tokens.length, module: null };
}

/**
 * extractFromSource returns the message templates in one file and the modules it declares as
 * `#[cfg(test)] mod name;` ({ name, path }, path from #[path] or null). Throws
 * RustExtractError on a template it must refuse.
 */
export function extractFromSource(source, file = "<source>") {
  const tokens = tokenize(source, file);
  const messages = [];
  const testModules = [];
  let wholeFileIsTest = false;

  let k = 0;
  while (k < tokens.length) {
    const token = tokens[k];

    if (token.value === "#" && token.type === "punct") {
      const inner = tokens[k + 1]?.value === "!";
      const open = inner ? k + 2 : k + 1;
      if (tokens[open]?.value === "[") {
        const end = closing(tokens, open);
        if (end === -1) break;
        if (isTestCfg(tokens.slice(open + 1, end))) {
          if (inner) {
            wholeFileIsTest = true;
            break;
          }
          const skipped = skipItem(tokens, end + 1);
          if (skipped.module !== null) testModules.push(skipped.module);
          k = skipped.next;
          continue;
        }
        k = end + 1;
        continue;
      }
    }

    if (token.type === "ident" && token.value === MACRO && tokens[k + 1]?.value === "!") {
      const open = tokens[k + 2];
      if (open?.type === "punct" && open.value in OPEN) {
        const end = closing(tokens, k + 2);
        if (end === -1) throw new RustExtractError(file, token.line, "unterminated message!");
        const args = tokens.slice(k + 3, end);
        messages.push({ message: templateOf(args, file, token.line), line: token.line });
        k = end + 1;
        continue;
      }
    }
    k += 1;
  }

  return wholeFileIsTest ? { messages: [], testModules: [], wholeFileIsTest } : { messages, testModules, wholeFileIsTest };
}

function templateOf(args, file, line) {
  const literal = args[0];
  if (args.length !== 1 || literal.type !== "string") {
    throw new RustExtractError(
      file,
      line,
      "message! takes exactly one string literal (no concat!, constants, variables or format arguments)",
    );
  }
  if (literal.kind !== "str" && literal.kind !== "raw-str") {
    throw new RustExtractError(file, line, "message! takes a string literal, not a byte or C string");
  }
  const template = literal.value;
  if (template !== template.trim().replace(/\s+/g, " ")) {
    throw new RustExtractError(
      file,
      line,
      "a message template has no leading, trailing or repeated whitespace and no newlines: " +
        "the template is its catalog key exactly as written",
    );
  }
  return template;
}

async function* sourceFiles(dir, extension) {
  let entries;
  try {
    entries = await readdir(dir, { withFileTypes: true });
  } catch (err) {
    if (err.code === "ENOENT") return;
    throw err;
  }
  for (const entry of entries.sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) yield* sourceFiles(full, extension);
    else if (entry.isFile() && entry.name.endsWith(extension)) yield full;
  }
}

// moduleDir is where the files of the modules `file` declares live: next to lib.rs, main.rs
// and mod.rs, and in a folder named after the file otherwise (src/formula.rs -> src/formula/).
function moduleDir(file) {
  const name = basename(file);
  if (name === "lib.rs" || name === "main.rs" || name === "mod.rs") return dirname(file);
  return join(dirname(file), name.slice(0, -".rs".length));
}

// rulesArea files a template under the crate's top-level module it comes from
// (src/formula/check.rs -> rules/formula), so translation can be batched by area.
function rulesArea(root, file) {
  const parts = relative(root, file).split(sep);
  const top = parts[0].replace(/\.(rs|tariff)$/, "");
  return top === "lib" ? "rules" : `rules/${top}`;
}

/**
 * extractRust returns one Go-scope and one client entry per message! template and standard
 * library doc template under `roots`, and the templates the string filter rejected.
 */
export async function extractRust(repoRoot, roots = RUST_ROOTS) {
  const entries = [];
  const rejected = [];

  for (const root of roots) {
    const absoluteRoot = join(repoRoot, root);
    const parsed = [];
    const excluded = [];
    for await (const file of sourceFiles(absoluteRoot, ".rs")) {
      const result = extractFromSource(await readFile(file, "utf8"), relative(repoRoot, file));
      parsed.push({ file, ...result });
      for (const module of result.testModules) {
        const target =
          module.path === null
            ? join(moduleDir(file), `${module.name}.rs`)
            : join(dirname(file), module.path);
        const base = target.replace(/\.rs$/, "");
        excluded.push(target, `${base}.rs`, join(base, "mod.rs"), `${base}${sep}`);
      }
    }

    const addTemplates = (path, messages, kind) => {
      const relPath = relative(repoRoot, path);
      for (const { message, line } of messages) {
        const reason = reject(message, { positional: true });
        if (reason !== null) {
          rejected.push({ value: message, file: relPath, line, kind, reason });
          continue;
        }
        const site = { message, file: relPath, line, kind };
        entries.push({ ...site, area: rulesArea(absoluteRoot, path), scope: "go" });
        entries.push({ ...site, area: RULES_CLIENT_AREA, scope: "ts" });
      }
    };

    for (const { file: path, messages } of parsed) {
      if (excluded.some((excludedPath) => path === excludedPath || path.startsWith(excludedPath))) {
        continue;
      }
      addTemplates(path, messages, "rules-message");
    }
    for await (const path of sourceFiles(absoluteRoot, ".tariff")) {
      const source = await readFile(path, "utf8");
      addTemplates(path, extractTariffDocs(source, relative(repoRoot, path)), "rules-doc");
    }
  }

  return { entries, rejected };
}
