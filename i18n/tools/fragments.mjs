// fragments.mjs finds sentences the catalog cannot translate because they are assembled in
// English rather than written whole.
//
// Two shapes have shipped and both read as English in every locale:
//
//   - a sentence split by markup:  {t("You've used")} <b>{pct}</b> {t("of this month's allowance")}
//     Each half is its own catalog entry, so no language can move the bold amount. Write it
//     once with rt("You've used <b>{0}</b> of this month's allowance", { b }, pct).
//   - English handed to a message: t("{0} {1}", n, n === 1 ? "stop" : "stops"),
//     t("Send {0}", pluralize("notice", n)). Use an ICU plural, or one message per case.
//
//   - English built in a template literal where a person reads it:
//     toast.success(`${n} notices sent`), title={`Remove ${name}`}, { label: `Line ${i}` }.
//     It never reaches the catalog at all. Use t("{0, plural, …}", n) / t("Remove {0}", name).
//
// Only shapes that are wrong wherever they appear are reported, so a finding is always a
// defect: an argument that is a variable may well hold translated text, and is left alone.
// A literal in a reported position that really is not prose for a person — a key, a code,
// a value the server parses — takes `i18n-ignore: <reason>` in a comment on or above it.
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { parse } from "@babel/parser";
import { SOURCE_ROOTS } from "./extract-ts.mjs";

const SKIP_DIR = new Set(["node_modules", "generated", "__tests__", "__snapshots__", "dist"]);
const SKIP_FILE = /\.(test|spec|stories)\.[jt]sx?$/;
const TRANSLATORS = new Set(["t", "translate", "rt", "translateRich"]);
// Helpers that turn an identifier or a count into English words.
const ENGLISH_HELPERS = new Set(["pluralize", "toTitleCase", "humanizeToolName", "capitalize"]);
// Where a person reads a string: the props and keys the extractor treats as text, less the
// ones that also carry data (`body`, `text`, `detail`, `header` are payloads as often as
// prose).
const READ_KEYS = new Set([
  "label", "description", "placeholder", "title", "subtitle", "heading", "caption",
  "loadingLabel", "loadingText", "hint", "help", "helper", "message", "successMessage",
  "errorMessage", "emptyMessage", "emptyLabel", "noResultsMessage", "searchPlaceholder",
  "confirmLabel", "confirmText", "cancelLabel", "submitLabel", "tooltip", "alt",
  "ariaLabel", "aria-label", "summary", "sub", "footer",
]);
const TOAST_METHODS = new Set(["success", "error", "info", "warning", "message", "loading", "promise"]);
const IGNORE = /i18n-ignore:\s*\S/;

// Inline elements that sit inside a sentence rather than between sentences.
const INLINE = /^(strong|b|em|i|kbd|Kbd|code|span|a|Link|mark|u|abbr|time)$/;

async function* walk(dir) {
  let entries;
  try {
    entries = await readdir(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of entries) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (!SKIP_DIR.has(entry.name)) yield* walk(full);
    } else if (/\.[jt]sx?$/.test(entry.name) && !SKIP_FILE.test(entry.name)) {
      yield full;
    }
  }
}

function visit(node, fn) {
  if (node === null || typeof node !== "object") return;
  if (Array.isArray(node)) {
    for (const child of node) visit(child, fn);
    return;
  }
  if (typeof node.type !== "string") return;
  fn(node);
  for (const key of Object.keys(node)) {
    if (key === "loc" || key.endsWith("Comments")) continue;
    visit(node[key], fn);
  }
}

function visitWithParents(node, chain, fn) {
  if (node === null || typeof node !== "object") return;
  if (Array.isArray(node)) {
    for (const child of node) visitWithParents(child, chain, fn);
    return;
  }
  if (typeof node.type !== "string") return;
  fn(node, chain);
  const next = [node, ...chain].slice(0, 6);
  for (const key of Object.keys(node)) {
    if (key === "loc" || key.endsWith("Comments")) continue;
    visitWithParents(node[key], next, fn);
  }
}

const hasWords = (text) => /\p{L}{2,}/u.test(text);

function translatorName(call) {
  return call.type === "CallExpression" && call.callee.type === "Identifier" ? call.callee.name : null;
}

// A child that carries words of the sentence: text, or {t("...")}.
function isWordy(child) {
  if (child.type === "JSXText") return hasWords(child.value);
  if (child.type !== "JSXExpressionContainer") return false;
  const expr = child.expression;
  return TRANSLATORS.has(translatorName(expr)) && expr.arguments[0]?.type === "StringLiteral";
}

function elementName(element) {
  const name = element.openingElement.name;
  return name.name ?? name.property?.name ?? "";
}

/** splitSentence reports a JSX element whose words are cut in two by an inline element. */
function splitSentence(element) {
  const kids = element.children;
  const wordy = kids.flatMap((child, index) => (isWordy(child) ? [index] : []));
  for (let i = 0; i < wordy.length - 1; i++) {
    const between = kids.slice(wordy[i] + 1, wordy[i + 1]);
    const inline = between.filter((c) => c.type === "JSXElement");
    if (inline.length > 0 && inline.every((c) => INLINE.test(elementName(c)))) {
      // Only a sentence the catalog already half-holds: one side is a t() call.
      const pair = [kids[wordy[i]], kids[wordy[i + 1]]];
      if (pair.some((c) => c.type === "JSXExpressionContainer")) return true;
    }
  }
  return false;
}

/** englishArgument returns the English an argument passes into a message, or null. */
function englishArgument(expr) {
  if (!expr) return null;
  switch (expr.type) {
    case "StringLiteral":
      // A wire value ("SHIPMENT_ID") or a template token ("{loginSlug}") is data, not prose.
      if (/^[A-Z0-9_]+$/.test(expr.value) || /^\{[\w.]+\}$/.test(expr.value)) return null;
      return /\p{L}{3,}/u.test(expr.value) ? expr.value : null;
    case "TemplateLiteral": {
      const text = expr.quasis.map((q) => q.value.cooked).join("");
      return /\p{L}{3,}/u.test(text) ? `\`${text}\`` : null;
    }
    case "ConditionalExpression":
      return englishArgument(expr.consequent) ?? englishArgument(expr.alternate);
    case "LogicalExpression":
      return englishArgument(expr.right) ?? englishArgument(expr.left);
    case "CallExpression": {
      const name = expr.callee.type === "Identifier" ? expr.callee.name : null;
      return name !== null && ENGLISH_HELPERS.has(name) ? `${name}(…)` : null;
    }
    default:
      return null;
  }
}

/** prose returns the words a template literal writes, or null when it writes no sentence. */
function prose(template) {
  const text = template.quasis.map((q) => q.value.cooked ?? "").join(" ");
  const words = text.match(/\p{L}{2,}/gu) ?? [];
  if (words.length < 2) return null;
  // A class list, path or key: no spaces between its words, or utility-shaped tokens.
  if (!/\p{L}{2,}[^\S\n]+\p{L}/u.test(text)) return null;
  if (/(^|\s)[a-z]+(-[a-z0-9]+)+(\s|$)/.test(text) && !/[.,!?;:]\s/.test(text)) return null;
  return text.replace(/\s+/g, " ").trim();
}

function keyName(key) {
  if (!key) return null;
  return key.type === "Identifier" ? key.name : key.type === "StringLiteral" ? key.value : null;
}

/**
 * readPosition says whether a template literal sits where a person reads it, by its parent
 * chain: a toast argument or toast option, a JSX child, a text attribute, or a text-keyed
 * property. A literal inside a translator call is already a message argument and is left to
 * the other rules.
 */
function readPosition(chain) {
  const [parent, grand, great] = chain;
  if (!parent) return null;
  if (parent.type === "JSXExpressionContainer") {
    if (grand?.type === "JSXElement" || grand?.type === "JSXFragment") return "jsx-child";
    if (grand?.type === "JSXAttribute") {
      const name = grand.name.type === "JSXIdentifier" ? grand.name.name : null;
      return name !== null && READ_KEYS.has(name) ? `${name}=` : null;
    }
    return null;
  }
  if (parent.type === "ObjectProperty" && READ_KEYS.has(keyName(parent.key))) {
    // An option on a toast, or a text key anywhere: both are read.
    return `${keyName(parent.key)}:`;
  }
  if (parent.type === "CallExpression") {
    const callee = parent.callee;
    if (
      callee.type === "MemberExpression" &&
      callee.object.type === "Identifier" &&
      callee.object.name === "toast" &&
      TOAST_METHODS.has(callee.property.name)
    ) {
      return `toast.${callee.property.name}`;
    }
    if (callee.type === "Identifier" && callee.name === "toast") return "toast";
  }
  // `cond ? `…` : `…`` and `a ?? `…`` in a read position.
  if (parent.type === "ConditionalExpression" || parent.type === "LogicalExpression") {
    return readPosition([grand, great, ...chain.slice(3)]);
  }
  return null;
}

function ignored(node, lines) {
  const line = node.loc.start.line;
  return IGNORE.test(lines[line - 1] ?? "") || IGNORE.test(lines[line - 2] ?? "");
}

/**
 * findFragments scans source roots and returns every split sentence and every English
 * argument, each as { file, line, kind, detail }.
 */
export async function findFragments(repoRoot, roots = SOURCE_ROOTS) {
  const findings = [];
  for (const root of roots) {
    for await (const file of walk(join(repoRoot, root))) {
      const source = await readFile(file, "utf8");
      let ast;
      try {
        ast = parse(source, { sourceType: "module", plugins: ["typescript", "jsx"], errorRecovery: true });
      } catch {
        continue;
      }
      const rel = relative(repoRoot, file);
      const lines = source.split("\n");
      visitWithParents(ast.program, [], (node, chain) => {
        if (node.type !== "TemplateLiteral") return;
        const text = prose(node);
        if (text === null) return;
        const where = readPosition(chain);
        if (where === null || ignored(node, lines)) return;
        findings.push({ file: rel, line: node.loc.start.line, kind: "untranslated-template", detail: `${where} \`${text}\`` });
      });
      visit(ast.program, (node) => {
        if (node.type === "JSXElement" && splitSentence(node)) {
          findings.push({ file: rel, line: node.loc.start.line, kind: "split-sentence", detail: elementName(node) });
          return;
        }
        const name = translatorName(node);
        if (!TRANSLATORS.has(name)) return;
        const skip = name === "rt" || name === "translateRich" ? 2 : 1;
        for (const arg of node.arguments.slice(skip)) {
          const english = englishArgument(arg);
          if (english !== null) {
            findings.push({ file: rel, line: node.loc.start.line, kind: "english-argument", detail: english });
          }
        }
      });
    }
  }
  return findings;
}

export function formatFinding({ file, line, kind, detail }) {
  switch (kind) {
    case "split-sentence":
      return `${file}:${line}  a sentence is split by <${detail}>; write it whole with rt()`;
    case "untranslated-template":
      return `${file}:${line}  ${detail} builds English outside the catalog; use t()/translate() with placeholders`;
    default:
      return `${file}:${line}  ${JSON.stringify(detail)} is English passed into a message; use a plural or one message per case`;
  }
}
