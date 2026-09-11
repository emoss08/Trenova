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

const DEP_HOOKS = new Set([
  "useCallback",
  "useMemo",
  "useEffect",
  "useLayoutEffect",
  "useInsertionEffect",
  "useImperativeHandle",
]);

// dependencyArray returns the literal dependency list of a React hook call, which is what
// has to gain `t` when a translated string is introduced inside the hook body. Without it
// the memoized value keeps the text of whatever language was active when it was built, and
// a language switch leaves stale strings on screen.
function dependencyArray(node) {
  if (node.type !== "CallExpression") return null;
  const callee = node.callee;
  const name =
    callee.type === "Identifier"
      ? callee.name
      : callee.type === "MemberExpression" && callee.property.type === "Identifier"
        ? callee.property.name
        : null;
  if (name === null || !DEP_HOOKS.has(name)) return null;

  const last = node.arguments[node.arguments.length - 1];
  return last !== undefined && last.type === "ArrayExpression" ? last : null;
}

function visit(node, parent, fn, stack, deps) {
  if (node === null || typeof node !== "object") return;
  if (Array.isArray(node)) {
    for (const child of node) visit(child, parent, fn, stack, deps);
    return;
  }
  if (typeof node.type !== "string") return;

  const isFunction =
    node.type === "FunctionDeclaration" ||
    node.type === "FunctionExpression" ||
    node.type === "ArrowFunctionExpression";

  const depList = dependencyArray(node);

  if (isFunction) stack.push(node);
  if (depList !== null) deps.push(depList);
  fn(node, parent, stack, deps);

  for (const key of Object.keys(node)) {
    if (key === "loc" || key === "leadingComments" || key === "trailingComments") continue;
    visit(node[key], node, fn, stack, deps);
  }

  if (depList !== null) deps.pop();
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
  const depArraysNeedingT = new Set();

  const callFor = (stack, deps) => {
    for (let i = stack.length - 1; i >= 0; i--) {
      if (isHookContext(stack[i], source)) {
        hookTargets.add(stack[i]);
        if (deps.length > 0) depArraysNeedingT.add(deps[deps.length - 1]);
        return "t";
      }
    }
    needsTranslate = true;
    return "translate";
  };

  visit(ast.program, null, (node, parent, stack, deps) => {
    switch (node.type) {
      case "JSXElement":
      case "JSXFragment": {
        for (const run of foldableRuns(node.children)) {
          if (reject(run.message, {}) !== null) continue;
          for (const child of run.nodes) {
            if (child.type === "JSXText") consumed.add(child);
          }
          const call = callFor(stack, deps);
          const args = run.expressions.map((container) =>
            source.slice(container.start + 1, container.end - 1).trim(),
          );
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

        const call = callFor(stack, deps);
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

        const call = callFor(stack, deps);
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
            const call = callFor(stack, deps);
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
            const call = callFor(stack, deps);
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
  }, [], []);

  if (edits.length === 0) return { changed: false, skipped };

  // A concise arrow has no block to declare the binding in. Falling back to the module-level
  // translate here would be wrong — inside a component it would freeze the text in whatever
  // language first rendered — so the arrow is given a body instead.
  const hookInserts = [];
  if (!alreadyHasHook) {
    for (const fn of hookTargets) {
      if (fn.body.type === "BlockStatement") {
        // A directive prologue ("use client", "use no memo") is only a directive while it
        // is the first statement. Inserting above it silently demotes it to a bare string
        // expression and turns the opt-out it encodes off.
        const directives = fn.body.directives ?? [];
        const after =
          directives.length > 0 ? directives[directives.length - 1].end : fn.body.start + 1;

        hookInserts.push({ start: after, end: after, text: "\n  const t = useT();\n" });
        continue;
      }

      // The body offsets sit inside any wrapping parentheses, so the block has to be
      // opened outside them: `=> (<svg/>)` becomes `=> { ...; return (<svg/>); }`.
      // Inserting rather than replacing leaves the edits inside the body untouched.
      const [open, close] = expandParens(source, fn.body.start, fn.body.end);
      hookInserts.push({ start: open, end: open, text: "{\n  const t = useT();\n  return " });
      hookInserts.push({ start: close, end: close, text: ";\n}" });
    }
  }

  const depInserts = [];
  for (const array of depArraysNeedingT) {
    const already = array.elements.some(
      (el) => el !== null && el.type === "Identifier" && el.name === "t",
    );
    if (already) continue;
    const insertAt = array.end - 1;
    // A dependency list written across lines usually ends with a trailing comma. Adding
    // ", t" after one produces the sparse array [a, , t], whose hole is a real bug and not
    // just a lint complaint.
    let back = insertAt - 1;
    while (back >= 0 && /\s/.test(source[back])) back -= 1;
    const needsComma = array.elements.length > 0 && source[back] !== ",";

    depInserts.push({
      start: insertAt,
      end: insertAt,
      text: needsComma ? ", t" : "t",
    });
  }

  const importInserts = [];
  const importAnchor = firstImportOffset(source);
  const imports = [];
  if (hookTargets.size > 0 && !alreadyImportsHook) imports.push(HOOK_IMPORT);
  if (needsTranslate && !alreadyImportsTranslate) imports.push(TRANSLATE_IMPORT);
  if (imports.length > 0) {
    importInserts.push({ start: importAnchor, end: importAnchor, text: `${imports.join("\n")}\n` });
  }

  const all = [...edits, ...hookInserts, ...depInserts, ...importInserts].sort(
    (a, b) => b.start - a.start,
  );

  let out = source;
  for (const edit of all) {
    out = out.slice(0, edit.start) + edit.text + out.slice(edit.end);
  }

  return { changed: out !== source, output: out, count: edits.length, skipped };
}

// expandParens widens a range over any balanced parentheses that wrap it, so a concise
// arrow body written across lines is treated as the single expression it is.
function expandParens(source, start, end) {
  let from = start;
  let to = end;

  for (;;) {
    let before = from - 1;
    while (before >= 0 && /\s/.test(source[before])) before -= 1;

    let after = to;
    while (after < source.length && /\s/.test(source[after])) after += 1;

    if (before >= 0 && source[before] === "(" && after < source.length && source[after] === ")") {
      from = before;
      to = after + 1;
      continue;
    }

    return [from, to];
  }
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
