/**
 * The audit behind `pnpm lint:skeletons` (scripts/check-loading-skeletons.mjs): it reads
 * every component under each app's and each package's `src` in a client root and returns each
 * loading state that does not look like what loads. The rules are described there.
 */

import { existsSync, readFileSync } from "node:fs";
import { glob } from "node:fs/promises";
import { createRequire } from "node:module";
import { dirname, join, relative, resolve } from "node:path";

const require = createRequire(new URL("../apps/web/package.json", import.meta.url));
const { parseSync } = require("oxc-parser");

const IGNORE = "loading-skeleton-ignore:";
const EXTENSIONS = [".tsx", ".ts", "/index.tsx", "/index.ts"];
const SPINNERS = new Set(["ComponentLoader", "Spinner"]);
const DATA_TABLE_MODULE = /(?:^|\/)data-table\/data-table$|^\.\.?\/data-table$/;

function position(parsed, offset) {
  let low = 0;
  let high = parsed.lineStarts.length - 1;
  while (low < high) {
    const middle = (low + high + 1) >> 1;
    if (parsed.lineStarts[middle] <= offset) {
      low = middle;
    } else {
      high = middle - 1;
    }
  }
  return { line: low, column: offset - parsed.lineStarts[low] };
}

function hasIgnore(parsed, node) {
  const { line } = position(parsed, node.start);
  const lines = parsed.text.split("\n");
  return [lines[line], lines[line - 1]].some((text) => text?.includes(IGNORE));
}

function jsxName(name) {
  if (name.type === "JSXIdentifier") {
    return name.name;
  }
  if (name.type === "JSXMemberExpression") {
    return `${jsxName(name.object)}.${name.property.name}`;
  }
  if (name.type === "JSXNamespacedName") {
    return `${name.namespace.name}:${name.name.name}`;
  }
  return "";
}

function tagName(element) {
  return jsxName(element.openingElement.name);
}

function attribute(element, name) {
  return element.openingElement.attributes.find(
    (entry) => entry.type === "JSXAttribute" && entry.name.name === name,
  );
}

/** The JSX elements a child expression can draw: through `&&`, ternaries and fragments. */
function drawnElements(node, into = []) {
  if (!node) {
    return into;
  }
  switch (node.type) {
    case "JSXElement":
      into.push(node);
      break;
    case "JSXFragment":
      for (const child of node.children) {
        drawnElements(child, into);
      }
      break;
    case "JSXExpressionContainer":
    case "ParenthesizedExpression":
      drawnElements(node.expression, into);
      break;
    case "LogicalExpression":
      drawnElements(node.right, into);
      break;
    case "ConditionalExpression":
      drawnElements(node.consequent, into);
      drawnElements(node.alternate, into);
      break;
    default:
      break;
  }
  return into;
}

function unwrap(node) {
  let current = node;
  while (current?.type === "ParenthesizedExpression" || current?.type === "TSAsExpression") {
    current = current.expression;
  }
  return current;
}

/** What an arrow or function expression returns. */
function returned(node) {
  const fn = unwrap(node);
  if (fn?.type !== "ArrowFunctionExpression" && fn?.type !== "FunctionExpression") {
    return null;
  }
  if (fn.body.type !== "BlockStatement") {
    return unwrap(fn.body);
  }
  const statement = fn.body.body.find((entry) => entry.type === "ReturnStatement");
  return unwrap(statement?.argument);
}

/** The module behind `import("…")`, through any `.then((m) => ({ default: m.X }))`. */
function dynamicImportSpecifier(node) {
  let call = unwrap(node);
  if (
    call?.type === "CallExpression" &&
    call.callee.type === "MemberExpression" &&
    call.callee.property.name === "then"
  ) {
    call = unwrap(call.callee.object);
  }
  if (call?.type === "ImportExpression" && typeof call.source.value === "string") {
    return call.source.value;
  }
  return null;
}

/** `lazy(() => import(...))` and `lazyComponent(() => import(...), pick)`. */
function lazySpecifier(init) {
  const call = unwrap(init);
  if (call?.type !== "CallExpression") {
    return null;
  }
  const callee =
    call.callee.type === "Identifier"
      ? call.callee.name
      : call.callee.type === "MemberExpression"
        ? call.callee.property.name
        : "";
  if (callee !== "lazy" && callee !== "lazyComponent") {
    return null;
  }
  return dynamicImportSpecifier(returned(call.arguments[0]));
}

function* topLevelStatements(program) {
  for (const statement of program.body) {
    if (
      (statement.type === "ExportNamedDeclaration" ||
        statement.type === "ExportDefaultDeclaration") &&
      statement.declaration
    ) {
      yield statement.declaration;
    } else {
      yield statement;
    }
  }
}

/**
 * Audits a client root and returns its violations, each with the file relative to the root,
 * a 1-based line and column, the rule and the message.
 */
export async function auditLoadingSkeletons(root) {
  const aliases = [
    { prefix: "@trenova/shared/", root: join(root, "packages/shared/src") },
    { prefix: "@trenova/graphql/", root: join(root, "packages/graphql/src") },
  ];
  const moduleCache = new Map();

  /** A parsed module: its text, its program, and the offset each line starts at. */
  function parse(file) {
    let parsed = moduleCache.get(file);
    if (!parsed) {
      const text = readFileSync(file, "utf8");
      const { program } = parseSync(file, text, { lang: "tsx" });
      const lineStarts = [0];
      for (let index = 0; index < text.length; index += 1) {
        if (text.charCodeAt(index) === 10) {
          lineStarts.push(index + 1);
        }
      }
      parsed = { file, text, program, lineStarts };
      moduleCache.set(file, parsed);
    }
    return parsed;
  }

  function appSourceRoot(file) {
    const match = /^(.*\/apps\/[^/]+\/src)\//.exec(file);
    return match ? match[1] : join(root, "packages/shared/src");
  }

  function resolveModule(specifier, fromFile) {
    let base = null;
    if (specifier.startsWith(".")) {
      base = resolve(dirname(fromFile), specifier);
    } else if (specifier.startsWith("@/")) {
      base = join(appSourceRoot(fromFile), specifier.slice(2));
    } else {
      const alias = aliases.find((entry) => specifier.startsWith(entry.prefix));
      if (alias) {
        base = join(alias.root, specifier.slice(alias.prefix.length));
      }
    }
    if (!base) {
      return null;
    }
    if (/\.tsx?$/.test(base) && existsSync(base)) {
      return base;
    }
    for (const extension of EXTENSIONS) {
      if (existsSync(base + extension)) {
        return base + extension;
      }
    }
    return null;
  }

  /** Follows a JSX tag to the file that defines it, or null when it cannot be followed. */
  function resolveTag(name, parsed) {
    const [head, member] = name.split(".");
    for (const statement of topLevelStatements(parsed.program)) {
      if (statement.type === "ImportDeclaration") {
        if (statement.specifiers.some((specifier) => specifier.local.name === head)) {
          return resolveModule(statement.source.value, parsed.file);
        }
      }
      if (statement.type === "VariableDeclaration") {
        for (const declaration of statement.declarations) {
          if (declaration.id.type !== "Identifier" || declaration.id.name !== head) {
            continue;
          }
          const init = unwrap(declaration.init);
          if (!member || member === "Component") {
            const specifier = lazySpecifier(init);
            if (specifier) {
              return resolveModule(specifier, parsed.file);
            }
          }
          if (init?.type === "MemberExpression" && init.property.name === "Component") {
            return resolveTag(`${init.object.name}.Component`, parsed);
          }
          return parsed.file;
        }
      }
      if (
        (statement.type === "FunctionDeclaration" || statement.type === "ClassDeclaration") &&
        statement.id?.name === head
      ) {
        return parsed.file;
      }
    }
    return null;
  }

  /** Whether a module draws a `DataTable`, directly or through a module it re-exports. */
  function rendersDataTable(file, seen = new Set()) {
    if (!file || seen.has(file)) {
      return false;
    }
    seen.add(file);
    const parsed = parse(file);
    let imported = false;
    const reexports = [];
    for (const statement of parsed.program.body) {
      if (
        statement.type === "ImportDeclaration" &&
        DATA_TABLE_MODULE.test(statement.source.value) &&
        statement.specifiers.some(
          (specifier) =>
            specifier.type === "ImportSpecifier" && specifier.local.name === "DataTable",
        )
      ) {
        imported = true;
      }
      if (
        (statement.type === "ExportNamedDeclaration" ||
          statement.type === "ExportAllDeclaration") &&
        statement.source
      ) {
        reexports.push(resolveModule(statement.source.value, file));
      }
    }
    if (imported && /<DataTable[\s/<>]/.test(parsed.text)) {
      return true;
    }
    return reexports.some((target) => rendersDataTable(target, seen));
  }

  const violations = [];

  function report(parsed, node, rule, message) {
    if (hasIgnore(parsed, node)) {
      return;
    }
    const { line, column } = position(parsed, node.start);
    violations.push({
      file: relative(root, parsed.file),
      line: line + 1,
      column: column + 1,
      rule,
      message,
    });
  }

  function checkElement(parsed, node) {
    const name = tagName(node);
    if ((name === "Suspense" || name === "React.Suspense") && !attribute(node, "fallback")) {
      report(
        parsed,
        node,
        "suspense-without-fallback",
        "`<Suspense>` without a fallback draws nothing while it loads. Give it the skeleton of what loads, or `fallback={null}` when nothing on screen is right.",
      );
    }
    const fallback = attribute(node, "fallback");
    if (fallback?.value) {
      for (const drawn of drawnElements(fallback.value)) {
        const drawnName = tagName(drawn);
        if (SPINNERS.has(drawnName)) {
          report(
            parsed,
            drawn,
            "spinner-fallback",
            `\`<${drawnName}>\` as the fallback of \`<${name}>\` puts a spinner where the content lands, and the layout jumps when it arrives. Draw the skeleton of what loads.`,
          );
        }
      }
    }
    if (name !== "DataTableLazyComponent") {
      return;
    }
    for (const child of node.children.flatMap((entry) => drawnElements(entry))) {
      const childName = tagName(child);
      const target = resolveTag(childName, parsed);
      if (!target) {
        report(
          parsed,
          child,
          "unresolved-lazy-child",
          `\`<${childName}>\` inside \`<DataTableLazyComponent>\` could not be followed to its module, so nothing can say it is a table. Import it, or load it with \`lazy(() => import(...))\` or \`lazyComponent(...)\`.`,
        );
      } else if (!rendersDataTable(target)) {
        report(
          parsed,
          child,
          "table-skeleton-on-non-table",
          `\`<${childName}>\` (${relative(root, target)}) does not render a DataTable, so \`<DataTableLazyComponent>\` opens it as a grid of rows it never draws. Use \`<LazyComponent fallback={<…Skeleton />}>\` with a skeleton of its own layout.`,
        );
      }
    }
  }

  function visit(parsed, node) {
    if (!node || typeof node !== "object") {
      return;
    }
    if (Array.isArray(node)) {
      for (const entry of node) {
        visit(parsed, entry);
      }
      return;
    }
    if (node.type === "JSXElement") {
      checkElement(parsed, node);
    }
    for (const key in node) {
      if (key !== "parent") {
        const value = node[key];
        if (value && typeof value === "object") {
          visit(parsed, value);
        }
      }
    }
  }

  const files = [];
  for await (const file of glob(["apps/*/src/**/*.tsx", "packages/*/src/**/*.tsx"], {
    cwd: root,
    exclude: (name) => name === "node_modules" || name === "generated",
  })) {
    if (/\.(test|stories)\.tsx$/.test(file) || file.includes("__tests__")) {
      continue;
    }
    files.push(join(root, file));
  }

  for (const file of files) {
    const text = readFileSync(file, "utf8");
    if (!/DataTableLazyComponent|Suspense|fallback=/.test(text)) {
      continue;
    }
    visit(parse(file), parse(file).program);
  }

  return { violations, files: files.length };
}
