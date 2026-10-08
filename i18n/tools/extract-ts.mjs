// extract-ts.mjs finds user-facing text in the React apps by parsing them, not by grepping.
//
// A regex over JSX cannot tell `label="Save"` from `name="save"`, cannot see that a string
// sits inside a toast call, and cannot skip a `d="M12 2L2 7"` SVG path. Every one of those
// distinctions decides whether a string belongs in the catalog, so this walks the Babel AST
// and keys on syntactic position, exactly like the Go extractor does.
import { readdir, readFile } from "node:fs/promises";
import { join, relative, sep } from "node:path";
import { parse } from "@babel/parser";
import { foldableRuns } from "./fold.mjs";
import { reject, TEXT_PROPS } from "./filter.mjs";

export const SOURCE_ROOTS = [
  "client/apps/web/src",
  "client/apps/dash/src",
  "client/packages/shared/src",
];

const SKIP_DIR = new Set(["node_modules", "generated", "__tests__", "__snapshots__", "dist"]);
const SKIP_FILE = /\.(test|spec|stories)\.[jt]sx?$/;

const TRANSLATE_CALLEES = new Set(["t", "translate", "rt", "translateRich"]);

export const LABEL_MAP_CALLEE = "defineLabels";

// Keys beyond TEXT_PROPS whose literal values are headings: a column header, the group a
// navigation entry is listed under.
const CAPTION_KEYS = new Set(["header", "group"]);

/** unwrapExpression strips `as`, `satisfies` and `!`, which change a value's type, not its text. */
export function unwrapExpression(node) {
  let current = node;
  while (
    current &&
    (current.type === "TSAsExpression" ||
      current.type === "TSSatisfiesExpression" ||
      current.type === "TSTypeAssertion" ||
      current.type === "TSNonNullExpression")
  ) {
    current = current.expression;
  }
  return current;
}

const PARSER_PLUGINS = ["typescript", "jsx", "decorators-legacy", "explicitResourceManagement"];

async function* walkFiles(dir) {
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

// visit walks every AST node, handing each one its parent so a JSXText knows which element
// it sits in and a string literal knows which attribute holds it.
function visit(node, parent, fn) {
  if (node === null || typeof node !== "object") return;
  if (Array.isArray(node)) {
    for (const child of node) visit(child, parent, fn);
    return;
  }
  if (typeof node.type !== "string") return;

  fn(node, parent);

  for (const key of Object.keys(node)) {
    if (key === "loc" || key === "leadingComments" || key === "trailingComments") continue;
    visit(node[key], node, fn);
  }
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

// featureArea buckets a file under its route or package folder so translation can be
// reviewed a screen at a time instead of as one undifferentiated list.
//
// The area also picks the runtime catalog a string ships in (bundles.mjs), so `routes/<dir>`
// is reserved for a folder under client/apps/web/src/routes: that is the unit the web app
// loads code, and therefore catalogs, by. A file sitting directly in that routes folder (the
// app layout) is part of the shell, and the driver portal's flat route files are its own.
export function featureArea(relPath) {
  const parts = relPath.split(sep);
  const srcIdx = parts.indexOf("src");
  if (srcIdx === -1 || parts.length <= srcIdx + 1) return relPath;

  const scope = parts.slice(1, 3).join("/");
  const top = parts[srcIdx + 1];
  if (scope === "apps/web" && top === "routes" && parts.length > srcIdx + 3) {
    return `routes/${parts[srcIdx + 2]}`;
  }
  return `${scope}/${top}`;
}

export async function extractTypeScript(repoRoot, roots = SOURCE_ROOTS) {
  const entries = [];
  const rejected = [];
  const errors = [];

  for (const root of roots) {
    for await (const file of walkFiles(join(repoRoot, root))) {
      const relPath = relative(repoRoot, file);
      const source = await readFile(file, "utf8");

      let ast;
      try {
        ast = parse(source, { sourceType: "module", plugins: PARSER_PLUGINS, errorRecovery: true });
      } catch (err) {
        errors.push({ file: relPath, message: err.message });
        continue;
      }

      const area = featureArea(relPath);
      // JSXText nodes already folded into a whole-element message.
      const consumed = new Set();

      const record = (value, node, kind, prop = null, positional = false) => {
        const line = node.loc ? node.loc.start.line : 0;
        const reason = reject(value, { prop, positional });
        if (reason !== null) {
          rejected.push({ value, file: relPath, line, kind, prop, reason });
          return;
        }
        entries.push({
          message: value.trim().replace(/\s+/g, " "),
          file: relPath,
          line,
          kind,
          area,
          scope: "ts",
        });
      };

      visit(ast.program, null, (node, parent) => {
        switch (node.type) {
          case "JSXElement":
          case "JSXFragment": {
            for (const run of foldableRuns(node.children)) {
              for (const child of run.nodes) {
                if (child.type === "JSXText") consumed.add(child);
              }
              record(run.message, run.nodes[0], "jsx-block");
            }
            return;
          }

          case "JSXText": {
            // JSX collapses surrounding whitespace, so a text node that is only layout
            // whitespace between elements carries nothing to translate.
            if (node.value.trim() === "") return;
            if (consumed.has(node)) return;
            record(node.value, node, "jsx-text");
            return;
          }

          case "JSXAttribute": {
            const name = attributeName(node);
            if (name === null || !TEXT_PROPS.has(name)) return;
            const literal = literalFromAttributeValue(node.value);
            if (literal === null) return;
            record(literal.value, literal, "jsx-prop", name);
            return;
          }

          case "ObjectProperty": {
            // Label maps — navigation entries, select options, status badges — are plain
            // object literals, usually at module scope. They are some of the most visible
            // text in the product, so they belong in the catalog; but they are deliberately
            // NOT wrapped by the codemod. A module-level const evaluates once at import, so
            // a translate() call there would freeze whatever language loaded first. The
            // English text stays in the data as the key, and the component that renders it
            // translates at render.
            if (node.key.type !== "Identifier" || node.value.type !== "StringLiteral") return;
            // `group` names the section a navigation entry is listed under, read as its heading.
            if (!TEXT_PROPS.has(node.key.name) && !CAPTION_KEYS.has(node.key.name)) return;
            record(node.value.value, node.value, "object-label", node.key.name);
            return;
          }

          case "CallExpression": {
            const callee = node.callee;

            // defineLabels({ OnTheJob: "On the job" }): every value is a caption by
            // construction, so a lone lowercase word ("minutes") is kept like any other.
            if (callee.type === "Identifier" && callee.name === LABEL_MAP_CALLEE) {
              const map = unwrapExpression(node.arguments[0]);
              if (map?.type !== "ObjectExpression") return;
              for (const prop of map.properties) {
                const value = prop.type === "ObjectProperty" ? unwrapExpression(prop.value) : null;
                if (value?.type === "StringLiteral") {
                  record(value.value, value, "label-map", null, true);
                }
              }
              return;
            }

            // toast.success("Saved") / toast.error("...", { description: "..." })
            if (
              callee.type === "MemberExpression" &&
              callee.object.type === "Identifier" &&
              callee.object.name === "toast"
            ) {
              for (const arg of node.arguments) {
                if (arg.type === "StringLiteral") {
                  record(arg.value, arg, "toast");
                } else if (arg.type === "ObjectExpression") {
                  for (const prop of arg.properties) {
                    if (
                      prop.type !== "ObjectProperty" ||
                      prop.key.type !== "Identifier" ||
                      prop.value.type !== "StringLiteral"
                    ) {
                      continue;
                    }
                    if (!TEXT_PROPS.has(prop.key.name)) continue;
                    record(prop.value.value, prop.value, "toast", prop.key.name);
                  }
                }
              }
              return;
            }

            // Already-migrated call sites: t("..."), translate("...") and rich text,
            // rt("... <b>{0}</b> ...", tags), whose whole sentence is one message.
            if (
              callee.type === "Identifier" &&
              TRANSLATE_CALLEES.has(callee.name) &&
              node.arguments.length > 0 &&
              node.arguments[0].type === "StringLiteral"
            ) {
              record(node.arguments[0].value, node.arguments[0], "t-call");
            }
            return;
          }

          default:
        }
      });
    }
  }

  return { entries, rejected, errors };
}
