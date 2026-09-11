// codemod.mjs wraps user-facing literals in t(), reusing the extractor's AST rules and
// filter so a string can never be wrapped in one pass and skipped in the other.
//
// Edits are spliced into the original source at node offsets rather than regenerating the
// file from the AST. Regenerating would reformat all ~1,500 files and bury the actual change
// in noise; splicing leaves every untouched line byte-identical, which is what makes a
// per-feature-area review possible at all.
//
// The rule for how a string reaches `t` is where most of the care is. Inside a component or
// a custom hook it is the useT() binding, so text re-renders on a language switch. Anywhere
// else — module scope, a plain helper, an object literal — there is no hook to call, so the
// module-level `translate` is imported instead. Getting that backwards either breaks the
// rules of hooks or leaves text frozen in the language it first rendered in.
import { readFile, writeFile } from "node:fs/promises";
import { readdir } from "node:fs/promises";
import { join, relative } from "node:path";
import { parse } from "@babel/parser";
import { foldableRuns } from "./fold.mjs";
import { reject, TEXT_PROPS } from "./filter.mjs";

const PARSER_PLUGINS = ["typescript", "jsx", "decorators-legacy", "explicitResourceManagement"];
const SKIP_DIR = new Set(["node_modules", "generated", "__tests__", "__snapshots__", "dist", "i18n"]);
const SKIP_FILE = /\.(test|spec|stories)\.[jt]sx?$/;

const HOOK_IMPORT = 'import { useT } from "@trenova/shared/i18n/use-t";';
const TRANSLATE_IMPORT = 'import { translate } from "@trenova/shared/i18n/runtime";';

export async function* walkFiles(dir) {
  let dirents;
  try {
    dirents = await readdir(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of dirents) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (SKIP_DIR.has(entry.name)) continue;
      yield* walkFiles(full);
    } else if (/\.[jt]sx?$/.test(entry.name) && !SKIP_FILE.test(entry.name)) {
      yield full;
    }
  }
}

function visit(node, parent, fn, stack) {
  if (node === null || typeof node !== "object") return;
  if (Array.isArray(node)) {
    for (const child of node) visit(child, parent, fn, stack);
    return;
  }
  if (typeof node.type !== "string") return;

  const isFunction =
    node.type === "FunctionDeclaration" ||
    node.type === "FunctionExpression" ||
    node.type === "ArrowFunctionExpression";

  if (isFunction) stack.push(node);
  fn(node, parent, stack);

  for (const key of Object.keys(node)) {
    if (key === "loc" || key === "leadingComments" || key === "trailingComments") continue;
    visit(node[key], node, fn, stack);
  }

  if (isFunction) stack.pop();
}

// functionName recovers the binding a function was declared under, since an arrow function
// carries its name on the variable declarator rather than on itself.
function functionName(fn, source) {
  if (fn.id?.name) return fn.id.name;
  const before = source.slice(Math.max(0, fn.start - 120), fn.start);
  const match = /(?:const|let|var|function)\s+([A-Za-z0-9_$]+)\s*(?::[^=]*)?=\s*$/.exec(before);
  return match ? match[1] : null;
}

// A hook binding is only legal inside a component or another hook. React's own rule — an
// uppercase name or a use-prefix — is exactly the test, so it is the one used here.
function isHookContext(fn, source) {
  const name = functionName(fn, source);
  if (name === null) return false;
  return /^[A-Z]/.test(name) || /^use[A-Z]/.test(name);
}

function quote(text) {
  return JSON.stringify(text);
}

function attributeName(attr) {
  if (attr.name.type === "JSXIdentifier") return attr.name.name;
  if (attr.name.type === "JSXNamespacedName") {
    return `${attr.name.namespace.name}:${attr.name.name.name}`;
  }
  return null;
}

function literalFromAttributeValue(value) {
  if (value === null || value === undefined) return null;
  if (value.type === "StringLiteral") return value;
  if (value.type === "JSXExpressionContainer" && value.expression.type === "StringLiteral") {
    return value.expression;
  }
  return null;
}

export function transformSource(source, filePath) {
  let ast;
  try {
    ast = parse(source, { sourceType: "module", plugins: PARSER_PLUGINS });
  } catch (err) {
    return { changed: false, skipped: [{ reason: "parse error", detail: err.message }] };
  }

  const edits = [];
  const skipped = [];
  // Functions that need a `const t = useT()` line, and whether the file needs `translate`.
  const hookTargets = new Set();
  let needsTranslate = false;
  const consumed = new Set();

  const alreadyHasHook = /\buseT\s*\(/.test(source);
  const alreadyImportsHook = source.includes("i18n/use-t");
  const alreadyImportsTranslate = source.includes("i18n/runtime");

  // callFor decides between the hook binding and the module function for one site.
  const callFor = (stack) => {
    for (let i = stack.length - 1; i >= 0; i--) {
      if (isHookContext(stack[i], source)) {
        hookTargets.add(stack[i]);
        return "t";
      }
    }
    needsTranslate = true;
    return "translate";
  };

  visit(ast.program, null, (node, parent, stack) => {
    switch (node.type) {
      case "JSXElement":
      case "JSXFragment": {
        for (const run of foldableRuns(node.children)) {
          if (reject(run.message, {}) !== null) continue;
          for (const child of run.nodes) {
            if (child.type === "JSXText") consumed.add(child);
          }
          const call = callFor(stack);
          const args = run.expressions.map((expr) => source.slice(expr.start, expr.end));
          edits.push({
            start: run.nodes[0].start,
            end: run.nodes[run.nodes.length - 1].end,
            text: `${run.leading}{${call}(${[quote(run.message), ...args].join(", ")})}${run.trailing}`,
          });
        }
        return;
      }

      case "JSXText": {
        if (node.value.trim() === "" || consumed.has(node)) return;
        const reason = reject(node.value, {});
        if (reason !== null) return;

        // JSX collapses surrounding whitespace into layout; only the text itself moves
        // inside the call, so the spacing between sibling elements is preserved exactly.
        const leading = node.value.match(/^\s*/)[0];
        const trailing = node.value.match(/\s*$/)[0];
        const text = node.value.trim().replace(/\s+/g, " ");

        const call = callFor(stack);
        edits.push({
          start: node.start,
          end: node.end,
          text: `${leading}{${call}(${quote(text)})}${trailing}`,
        });
        return;
      }

      case "JSXAttribute": {
        const name = attributeName(node);
        if (name === null || !TEXT_PROPS.has(name)) return;
        const literal = literalFromAttributeValue(node.value);
        if (literal === null) return;
        if (reject(literal.value, { prop: name }) !== null) return;

        const call = callFor(stack);
        edits.push({
          start: node.value.start,
          end: node.value.end,
          text: `{${call}(${quote(literal.value.trim().replace(/\s+/g, " "))})}`,
        });
        return;
      }

      case "CallExpression": {
        const callee = node.callee;
        if (
          callee.type !== "MemberExpression" ||
          callee.object.type !== "Identifier" ||
          callee.object.name !== "toast"
        ) {
          return;
        }

        for (const arg of node.arguments) {
          if (arg.type === "StringLiteral") {
            if (reject(arg.value, {}) !== null) continue;
            const call = callFor(stack);
            edits.push({
              start: arg.start,
              end: arg.end,
              text: `${call}(${quote(arg.value.trim().replace(/\s+/g, " "))})`,
            });
            continue;
          }
          if (arg.type !== "ObjectExpression") continue;
          for (const prop of arg.properties) {
            if (
              prop.type !== "ObjectProperty" ||
              prop.key.type !== "Identifier" ||
              prop.value.type !== "StringLiteral" ||
              !TEXT_PROPS.has(prop.key.name)
            ) {
              continue;
            }
            if (reject(prop.value.value, { prop: prop.key.name }) !== null) continue;
            const call = callFor(stack);
            edits.push({
              start: prop.value.start,
              end: prop.value.end,
              text: `${call}(${quote(prop.value.value.trim().replace(/\s+/g, " "))})`,
            });
          }
        }
        return;
      }

      default:
    }
  }, []);

  if (edits.length === 0) return { changed: false, skipped };

  // A hook binding cannot be added to a function that is not one, so those sites keep the
  // module-level call rather than being silently dropped.
  for (const fn of hookTargets) {
    if (fn.body.type !== "BlockStatement") {
      skipped.push({ reason: "concise arrow body", detail: filePath });
    }
  }

  const hookInserts = [];
  if (!alreadyHasHook) {
    for (const fn of hookTargets) {
      if (fn.body.type !== "BlockStatement") continue;
      hookInserts.push({ start: fn.body.start + 1, end: fn.body.start + 1, text: "\n  const t = useT();\n" });
    }
  }

  const importInserts = [];
  const importAnchor = firstImportOffset(source);
  const imports = [];
  if (hookTargets.size > 0 && !alreadyImportsHook) imports.push(HOOK_IMPORT);
  if (needsTranslate && !alreadyImportsTranslate) imports.push(TRANSLATE_IMPORT);
  if (imports.length > 0) {
    importInserts.push({ start: importAnchor, end: importAnchor, text: `${imports.join("\n")}\n` });
  }

  const all = [...edits, ...hookInserts, ...importInserts].sort((a, b) => b.start - a.start);

  let out = source;
  for (const edit of all) {
    out = out.slice(0, edit.start) + edit.text + out.slice(edit.end);
  }

  return { changed: out !== source, output: out, count: edits.length, skipped };
}

function firstImportOffset(source) {
  const match = /^import\s/m.exec(source);
  return match ? match.index : 0;
}

export async function runCodemod(repoRoot, targetDir, { dryRun }) {
  let files = 0;
  let changedFiles = 0;
  let replacements = 0;
  const skipped = [];

  for await (const file of walkFiles(join(repoRoot, targetDir))) {
    files += 1;
    const source = await readFile(file, "utf8");
    const result = transformSource(source, relative(repoRoot, file));

    for (const s of result.skipped ?? []) skipped.push({ ...s, file: relative(repoRoot, file) });
    if (!result.changed) continue;

    changedFiles += 1;
    replacements += result.count;
    if (!dryRun) await writeFile(file, result.output, "utf8");
  }

  return { files, changedFiles, replacements, skipped };
}
