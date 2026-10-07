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
// Only shapes that are wrong wherever they appear are reported, so a finding is always a
// defect: an argument that is a variable may well hold translated text, and is left alone.
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { parse } from "@babel/parser";
import { SOURCE_ROOTS } from "./extract-ts.mjs";

const SKIP_DIR = new Set(["node_modules", "generated", "__tests__", "__snapshots__", "dist"]);
const SKIP_FILE = /\.(test|spec|stories)\.[jt]sx?$/;
const TRANSLATORS = new Set(["t", "translate", "rt"]);
// Helpers that turn an identifier or a count into English words.
const ENGLISH_HELPERS = new Set(["pluralize", "toTitleCase", "humanizeToolName", "capitalize"]);
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
      visit(ast.program, (node) => {
        if (node.type === "JSXElement" && splitSentence(node)) {
          findings.push({ file: rel, line: node.loc.start.line, kind: "split-sentence", detail: elementName(node) });
          return;
        }
        const name = translatorName(node);
        if (!TRANSLATORS.has(name)) return;
        const skip = name === "rt" ? 2 : 1;
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
  return kind === "split-sentence"
    ? `${file}:${line}  a sentence is split by <${detail}>; write it whole with rt()`
    : `${file}:${line}  ${JSON.stringify(detail)} is English passed into a message; use a plural or one message per case`;
}
