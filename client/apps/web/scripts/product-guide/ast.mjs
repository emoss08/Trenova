/**
 * A small static evaluator over oxc's ESTree output.
 *
 * The product guide is built from the app's own configuration without running
 * it: the navigation config imports icons and the router imports every page,
 * and none of that belongs in a code generator. So literals are read straight
 * from the syntax tree, and anything that is not a literal — an icon, a
 * function — reads as UNKNOWN and is left out.
 */

import { readFileSync } from "node:fs";
import { parseSync } from "oxc-parser";

export const UNKNOWN = Symbol("unknown");

export function parseFile(file) {
  const source = readFileSync(file, "utf8");
  const result = parseSync(file, source, { sourceType: "module" });
  const fatal = result.errors.filter((error) => error.severity === "Error");
  if (fatal.length > 0) {
    throw new Error(`${file}: ${fatal.map((error) => error.message).join("; ")}`);
  }

  return { file, source, program: result.program };
}

/** Top-level `const name = …` and `export const name = …` initializers by name. */
export function topLevelConsts(program) {
  const found = new Map();
  for (const statement of program.body) {
    const declaration =
      statement.type === "ExportNamedDeclaration" ? statement.declaration : statement;
    if (declaration?.type !== "VariableDeclaration") {
      continue;
    }
    for (const declarator of declaration.declarations) {
      if (declarator.id.type === "Identifier" && declarator.init) {
        found.set(declarator.id.name, declarator.init);
      }
    }
  }

  return found;
}

function unwrap(node) {
  let current = node;
  while (
    current &&
    (current.type === "TSAsExpression" ||
      current.type === "TSSatisfiesExpression" ||
      current.type === "TSNonNullExpression" ||
      current.type === "ParenthesizedExpression")
  ) {
    current = current.expression;
  }

  return current;
}

export function propertyName(property) {
  if (property.computed) {
    return null;
  }
  if (property.key.type === "Identifier") {
    return property.key.name;
  }
  if (property.key.type === "Literal") {
    return String(property.key.value);
  }

  return null;
}

/**
 * Evaluates a literal-shaped node. `scope` resolves identifiers declared in
 * the same file; `members` resolves `Enum.Member` references such as
 * `Resource.Shipment`.
 */
export function evaluate(node, context) {
  const target = unwrap(node);
  if (!target) {
    return UNKNOWN;
  }

  switch (target.type) {
    case "Literal":
      return target.value;
    case "TemplateLiteral":
      return target.expressions.length === 0 ? target.quasis[0].value.cooked : UNKNOWN;
    case "ArrayExpression": {
      const values = [];
      for (const element of target.elements) {
        if (element === null) {
          continue;
        }
        if (element.type === "SpreadElement") {
          const spread = evaluate(element.argument, context);
          if (Array.isArray(spread)) {
            values.push(...spread);
          }
          continue;
        }
        values.push(evaluate(element, context));
      }
      return values;
    }
    case "ObjectExpression": {
      const value = {};
      for (const property of target.properties) {
        if (property.type === "SpreadElement") {
          const spread = evaluate(property.argument, context);
          if (spread && typeof spread === "object" && !Array.isArray(spread)) {
            Object.assign(value, spread);
          }
          continue;
        }
        const name = propertyName(property);
        if (name !== null) {
          value[name] = evaluate(property.value, context);
        }
      }
      return value;
    }
    case "Identifier": {
      if (target.name === "undefined") {
        return undefined;
      }
      const declared = context.scope?.get(target.name);
      if (declared && !context.resolving?.has(target.name)) {
        const resolving = new Set(context.resolving ?? []);
        resolving.add(target.name);
        return evaluate(declared, { ...context, resolving });
      }
      return UNKNOWN;
    }
    case "MemberExpression": {
      if (target.computed || target.object.type !== "Identifier") {
        return UNKNOWN;
      }
      const resolve = context.members?.[target.object.name];
      return resolve ? resolve(target.property.name) : UNKNOWN;
    }
    case "CallExpression": {
      // t("Label") is the label.
      if (
        target.callee.type === "Identifier" &&
        target.callee.name === "t" &&
        target.arguments.length >= 1
      ) {
        return evaluate(target.arguments[0], context);
      }
      return UNKNOWN;
    }
    case "UnaryExpression":
      if (target.operator === "!" && target.argument.type === "Literal") {
        return !target.argument.value;
      }
      return UNKNOWN;
    default:
      return UNKNOWN;
  }
}

/** Depth-first walk over every node, calling `visit(node, parent)`. */
export function walk(node, visit, parent = null) {
  if (!node || typeof node.type !== "string") {
    return;
  }
  visit(node, parent);
  for (const key of Object.keys(node)) {
    if (key === "parent") {
      continue;
    }
    const child = node[key];
    if (Array.isArray(child)) {
      for (const item of child) {
        if (item && typeof item.type === "string") {
          walk(item, visit, node);
        }
      }
    } else if (child && typeof child.type === "string") {
      walk(child, visit, node);
    }
  }
}

/** Removes UNKNOWN values so what reaches the catalog is only what was read. */
export function known(value) {
  if (value === UNKNOWN) {
    return undefined;
  }
  if (Array.isArray(value)) {
    return value.map(known).filter((item) => item !== undefined);
  }
  if (value && typeof value === "object") {
    const clean = {};
    for (const [key, item] of Object.entries(value)) {
      const kept = known(item);
      if (kept !== undefined) {
        clean[key] = kept;
      }
    }
    return clean;
  }

  return value;
}
