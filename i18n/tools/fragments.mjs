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
//   - a literal handed to a toast: toast.promise(p, { success: "Saved" }). The toast shows it
//     as written. Use t("Saved") in a component, translate("Saved") in a handler.
//
//   - a validation message written as a literal: z.string().min(1, "Name is required"),
//     .refine(fn, { message: "…" }). The form shows it exactly as written. Write
//     { error: () => translate("Name is required") }, which zod reads when it validates.
//     The same holds for a react-hook-form `rules={{ required: "…" }}` message.
//
//   - a module-level label map: const STATUS_LABELS = { InReview: "In review" }. Its values
//     never reach the catalog, and translate() there would freeze the language active at
//     import. Declare it with defineLabels({ ... }) from @trenova/shared/i18n/labels.
//
// Only shapes that are wrong wherever they appear are reported, so a finding is always a
// defect: an argument that is a variable may well hold translated text, and is left alone.
// A literal in a reported position that really is not prose for a person — a key, a code,
// a value the server parses — takes `i18n-ignore: <reason>` in a comment on or above it.
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { parse } from "@babel/parser";
import { LABEL_MAP_CALLEE, SOURCE_ROOTS, unwrapExpression } from "./extract-ts.mjs";
import { TEXT_PROPS } from "./filter.mjs";

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

// A caption: starts with a capital, has lowercase letters, and is either several words or
// one plain word, its first word made of letters ("EDI partners", "Speeding, reckless …"),
// with a lowercase letter somewhere. "In review" and "Coaching" are captions;
// "InReview", "ON_HOLD", "#fff", "size-4", "CON" and an SVG path are keys, codes and data.
const CAPTION = /^\p{Lu}[\p{L}'’&.-]*[,:;]?(\s+\S+)+$|^\p{Lu}\p{Ll}+$/u;

/**
 * labelMap reports a module-level object literal of string values that reads as captions:
 * at least two of them, and at least half of its values. When most of its keys are ones the
 * extractor already treats as text (`label`, `title`) it is a record of text fields instead,
 * collected as it is.
 */
function labelMap(init) {
  const object = unwrapExpression(init);
  if (object?.type !== "ObjectExpression" || object.properties.length < 2) return null;
  const values = [];
  let textKeys = 0;
  for (const prop of object.properties) {
    if (prop.type !== "ObjectProperty") return null;
    if (TEXT_PROPS.has(keyName(prop.key))) textKeys += 1;
    const value = unwrapExpression(prop.value);
    if (value.type !== "StringLiteral") return null;
    values.push(value.value);
  }
  // Mostly `label`/`title`/`description` keys: one record's text fields, which the
  // extractor collects as they are. A field-label map merely has a `description` field.
  if (textKeys * 2 >= values.length) return null;
  const captions = values.filter((value) => CAPTION.test(value.trim()) && /\p{Ll}/u.test(value));
  if (captions.length < 2 || captions.length * 2 < values.length) return null;
  return captions;
}

/**
 * wireValues says a declaration's type names its values as something other than text —
 * `Record<Action, PTOStatus>` maps one wire value to another — so captions-shaped values
 * ("Approved") are enum members, not labels.
 */
function wireValues(declarator) {
  const annotation = declarator.id.typeAnnotation?.typeAnnotation;
  if (annotation?.type !== "TSTypeReference") return false;
  const name = annotation.typeName.type === "Identifier" ? annotation.typeName.name : null;
  const params = annotation.typeParameters?.params ?? [];
  if (name !== "Record" && name !== "Readonly" && name !== "Partial") return false;
  const value = name === "Record" ? params[1] : null;
  if (value === null || value === undefined) return false;
  return value.type !== "TSStringKeyword";
}

function moduleDeclarations(program) {
  const declarators = [];
  for (const statement of program.body) {
    const declaration =
      statement.type === "ExportNamedDeclaration" ? statement.declaration : statement;
    if (declaration?.type !== "VariableDeclaration") continue;
    for (const declarator of declaration.declarations) {
      if (declarator.id.type === "Identifier" && declarator.init) {
        declarators.push({ statement, declarator });
      }
    }
  }
  return declarators;
}

/** frozenObject unwraps Object.freeze({ ... }), which freezes a map but translates nothing. */
function frozenObject(init) {
  const node = unwrapExpression(init);
  if (
    node?.type === "CallExpression" &&
    node.callee.type === "MemberExpression" &&
    node.callee.object.type === "Identifier" &&
    node.callee.object.name === "Object" &&
    node.callee.property.name === "freeze"
  ) {
    return node.arguments[0];
  }
  return node;
}

// zod checks whose first argument is a bound (min(1, …)) and whose message comes second, and
// checks whose message is the only argument (email("…")).
const ZOD_VALUE_FIRST = new Set([
  "min", "max", "length", "gt", "gte", "lt", "lte", "multipleOf", "regex", "startsWith",
  "endsWith", "includes", "refine", "size",
]);
const ZOD_MESSAGE_FIRST = new Set([
  "email", "url", "uuid", "nonempty", "int", "positive", "nonnegative", "negative",
  "nonpositive", "finite", "datetime", "date", "time", "base64", "jwt", "e164", "hostname",
  "lowercase", "uppercase", "nonoptional", "hex",
]);

function memberChainRoot(node) {
  let current = node;
  while (
    current.type === "MemberExpression" ||
    current.type === "CallExpression" ||
    current.type === "TSAsExpression" ||
    current.type === "TSNonNullExpression"
  ) {
    current =
      current.type === "MemberExpression"
        ? current.object
        : current.type === "CallExpression"
          ? current.callee
          : current.expression;
  }
  return current;
}

/**
 * schemaMessages returns the literal messages a zod call carries: an `error` or `message`
 * option, or a message passed in its place. Only a chain that starts at `z` or at a schema
 * (`…Schema`, `…Shape`) is a zod call; `ctx.addIssue({ message })` is one wherever it is.
 */
function schemaMessages(call) {
  if (call.callee.type !== "MemberExpression" || call.callee.computed) return [];
  const method = call.callee.property.name;
  const options = (arg) =>
    arg.type !== "ObjectExpression"
      ? []
      : arg.properties.filter(
          (prop) =>
            prop.type === "ObjectProperty" &&
            ["error", "message"].includes(keyName(prop.key)) &&
            prop.value.type === "StringLiteral",
        ).map((prop) => prop.value);
  if (method === "addIssue") return call.arguments.flatMap(options);
  const root = memberChainRoot(call.callee);
  if (root.type !== "Identifier" || !(root.name === "z" || /(schema|Schema|Shape)$/.test(root.name))) {
    return [];
  }
  const messageIndex = ZOD_VALUE_FIRST.has(method) ? 1 : ZOD_MESSAGE_FIRST.has(method) ? 0 : -1;
  return call.arguments.flatMap((arg, index) =>
    arg.type === "StringLiteral" ? (index === messageIndex ? [arg] : []) : options(arg),
  );
}

/**
 * ruleMessages returns the literal messages a react-hook-form `rules` object carries
 * (`rules={{ required: "…" }}`, `rules: { minLength: { value: 2, message: "…" } }`).
 */
function ruleMessages(node) {
  let rules = null;
  if (node.type === "JSXAttribute" && node.name.name === "rules") {
    rules = node.value?.type === "JSXExpressionContainer" ? node.value.expression : null;
  } else if (node.type === "ObjectProperty" && keyName(node.key) === "rules") {
    rules = node.value;
  }
  if (rules?.type !== "ObjectExpression") return [];
  return rules.properties.flatMap((prop) => {
    if (prop.type !== "ObjectProperty") return [];
    if (prop.value.type === "StringLiteral") return [prop.value];
    if (prop.value.type !== "ObjectExpression") return [];
    return prop.value.properties.filter(
      (inner) =>
        inner.type === "ObjectProperty" &&
        keyName(inner.key) === "message" &&
        inner.value.type === "StringLiteral",
    ).map((inner) => inner.value);
  });
}

/** toastLiterals returns the string literals a toast call shows as written. */
function toastLiterals(call) {
  const callee = call.callee;
  const toast =
    (callee.type === "Identifier" && callee.name === "toast") ||
    (callee.type === "MemberExpression" &&
      callee.object.type === "Identifier" &&
      callee.object.name === "toast");
  if (!toast) return [];
  // `error.message || "Failed"` and `ok ? "Saved" : "Failed"` show their literals too.
  const literals = (node) => {
    if (node.type === "StringLiteral") return [node];
    if (node.type === "LogicalExpression") return [...literals(node.left), ...literals(node.right)];
    if (node.type === "ConditionalExpression") {
      return [...literals(node.consequent), ...literals(node.alternate)];
    }
    if (node.type === "MemberExpression" && node.object.type === "ObjectExpression") {
      return literals(node.object);
    }
    if (node.type !== "ObjectExpression") return [];
    return node.properties
      .filter((prop) => prop.type === "ObjectProperty")
      .flatMap((prop) => literals(prop.value));
  };
  return call.arguments.flatMap(literals);
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
      for (const { statement, declarator } of moduleDeclarations(ast.program)) {
        const init = unwrapExpression(declarator.init);
        if (init.type === "CallExpression" && init.callee.name === LABEL_MAP_CALLEE) continue;
        if (wireValues(declarator)) continue;
        const captions = labelMap(frozenObject(init));
        if (captions === null || ignored(statement, lines) || ignored(declarator, lines)) continue;
        findings.push({
          file: rel,
          line: declarator.loc.start.line,
          kind: "untranslated-label-map",
          detail: `${declarator.id.name} (${captions.slice(0, 2).map((c) => JSON.stringify(c)).join(", ")}…)`,
        });
      }
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
        for (const literal of ruleMessages(node)) {
          if (ignored(literal, lines)) continue;
          findings.push({ file: rel, line: literal.loc.start.line, kind: "untranslated-schema-message", detail: literal.value });
        }
        if (node.type === "CallExpression") {
          for (const literal of toastLiterals(node)) {
            if (!hasWords(literal.value) || ignored(literal, lines)) continue;
            findings.push({ file: rel, line: literal.loc.start.line, kind: "untranslated-toast", detail: literal.value });
          }
          for (const literal of schemaMessages(node)) {
            if (ignored(literal, lines)) continue;
            findings.push({ file: rel, line: literal.loc.start.line, kind: "untranslated-schema-message", detail: literal.value });
          }
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
    case "untranslated-toast":
      return `${file}:${line}  ${JSON.stringify(detail)} is shown by a toast as written; use t() or translate()`;
    case "untranslated-schema-message":
      return `${file}:${line}  ${JSON.stringify(detail)} is a validation message the form shows as written; use { error: () => translate(…) }`;
    case "untranslated-label-map":
      return `${file}:${line}  ${detail} is a label map the catalog never sees; declare it with defineLabels()`;
    case "untranslated-template":
      return `${file}:${line}  ${detail} builds English outside the catalog; use t()/translate() with placeholders`;
    default:
      return `${file}:${line}  ${JSON.stringify(detail)} is English passed into a message; use a plural or one message per case`;
  }
}
