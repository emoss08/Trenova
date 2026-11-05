/**
 * The web app's module graph, read statically: which modules each module
 * imports (statically or with `import()`), the GraphQL documents it imports,
 * and the REST endpoints it calls. Nothing is imported or run.
 *
 * REST calls go through `api.<method>(path)` from `@trenova/shared/lib/api`.
 * Screens reach them through a service's own instance
 * (`export const userService = new UserService()`): a module that imports one
 * is not sent into the service's module, and each `userService.y` it names is
 * resolved to the one method it calls.
 */

import { existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";

import { parseFile, walk } from "../product-guide/ast.mjs";

const GRAPHQL_DOCUMENTS = "@trenova/graphql/generated/graphql";
const REST_METHODS = new Set(["get", "post", "put", "patch", "delete"]);
const EXTENSIONS = [".tsx", ".ts", "/index.tsx", "/index.ts"];

export function graphPaths(webRoot) {
  const src = join(webRoot, "src");
  const shared = join(webRoot, "..", "..", "packages", "shared", "src");

  return {
    src,
    shared,
    main: join(src, "main.tsx"),
    router: join(src, "router.tsx"),
  };
}

function withExtension(base) {
  if (/\.(tsx?|mjs|js)$/.test(base) && existsSync(base)) {
    return base;
  }
  for (const extension of EXTENSIONS) {
    if (existsSync(`${base}${extension}`)) {
      return `${base}${extension}`;
    }
  }

  return null;
}

/** The file a specifier names, or null for a package or an asset. */
export function resolveSpecifier(paths, specifier, from) {
  if (specifier.startsWith("@/")) {
    return withExtension(join(paths.src, specifier.slice(2)));
  }
  if (specifier.startsWith("@trenova/shared/")) {
    return withExtension(join(paths.shared, specifier.slice("@trenova/shared/".length)));
  }
  if (specifier.startsWith("./") || specifier.startsWith("../")) {
    return withExtension(resolve(dirname(from), specifier));
  }

  return null;
}

/**
 * A REST path as the route table spells it: every `${…}` a parameter, the
 * query string dropped.
 */
export function restPath(node) {
  let raw = null;
  if (node?.type === "Literal" && typeof node.value === "string") {
    raw = node.value;
  } else if (node?.type === "TemplateLiteral") {
    raw = node.quasis.map((quasi) => quasi.value.cooked).join(":param");
  }
  if (raw === null || !raw.startsWith("/")) {
    return null;
  }

  // `${id}${query}` is one parameter, and a parameter that does not start a
  // segment (`/roles/constraints${query}`) is a query string built elsewhere.
  return raw
    .split("?")[0]
    .replace(/(:param)+/g, ":param")
    .replace(/([^/]):param$/, "$1");
}

function restCall(node) {
  if (node.type !== "CallExpression" || node.callee.type !== "MemberExpression") {
    return null;
  }
  const { object, property } = node.callee;
  if (
    object.type !== "Identifier" ||
    object.name !== "api" ||
    property.type !== "Identifier" ||
    !REST_METHODS.has(property.name)
  ) {
    return null;
  }
  const path = restPath(node.arguments[0]);

  return path === null ? null : { method: property.name.toUpperCase(), path };
}

function instanceCall(node, instances) {
  if (
    node.type !== "MemberExpression" ||
    node.computed ||
    node.object.type !== "Identifier" ||
    node.property.type !== "Identifier"
  ) {
    return null;
  }
  const instance = instances.get(node.object.name);

  return instance ? { ...instance, method: node.property.name } : null;
}

export const ALL = Symbol("all");

function declaredNames(declaration) {
  if (!declaration) return [];
  if (
    (declaration.type === "FunctionDeclaration" || declaration.type === "ClassDeclaration") &&
    declaration.id
  ) {
    return [declaration.id.name];
  }
  if (declaration.type === "VariableDeclaration") {
    const names = [];
    for (const declarator of declaration.declarations) {
      walk(declarator.id, (node) => {
        if (node.type === "Identifier") names.push(node.name);
      });
    }
    return names;
  }
  return [];
}

function referencedNames(node) {
  const names = new Set();
  walk(node, (child) => {
    if (child.type === "Identifier") names.add(child.name);
  });
  return names;
}

/**
 * Which documents each top-level binding of a module that imports documents
 * uses, so a screen importing one wrapper from a module of many is held to
 * the operations that wrapper runs.
 */
function readBindings(program, info) {
  const anonymous = { documents: new Set(), locals: new Set() };
  const statements = [];
  for (const statement of program.body) {
    if (statement.type === "ImportDeclaration" || statement.type === "ExportAllDeclaration") {
      continue;
    }
    if (statement.type === "ExportNamedDeclaration" && !statement.declaration) {
      for (const spec of statement.specifiers ?? []) {
        if (!statement.source) {
          info.exports.set(spec.exported.name ?? spec.exported.value, spec.local.name);
        }
      }
      continue;
    }
    const declaration =
      statement.type === "ExportNamedDeclaration" || statement.type === "ExportDefaultDeclaration"
        ? statement.declaration
        : statement;
    const names = declaredNames(declaration);
    if (statement.type === "ExportNamedDeclaration") {
      for (const name of names) info.exports.set(name, name);
    }
    if (statement.type === "ExportDefaultDeclaration") {
      info.exports.set("default", names[0] ?? "default");
      if (names.length === 0) names.push("default");
    }
    statements.push({ names, node: declaration });
  }

  const topLevel = new Set(statements.flatMap((statement) => statement.names));
  for (const { names, node } of statements) {
    const referenced = referencedNames(node);
    const use = { documents: new Set(), locals: new Set() };
    for (const name of referenced) {
      if (info.documents.has(name)) use.documents.add(name);
      else if (topLevel.has(name) && !names.includes(name)) use.locals.add(name);
    }
    if (names.length === 0) {
      for (const name of use.documents) anonymous.documents.add(name);
      for (const name of use.locals) anonymous.locals.add(name);
      continue;
    }
    for (const name of names) info.bindings.set(name, use);
  }
  info.bindings.set(null, anonymous);
}

/** The documents a module's named exports use, or all of them. */
function documentsFor(info, names) {
  if (names === ALL || info.bindings.size === 0) {
    return info.documents;
  }
  const documents = new Set();
  const seen = new Set();
  const queue = [null, ...[...names].map((name) => info.exports.get(name) ?? name)];
  while (queue.length > 0) {
    const name = queue.pop();
    if (seen.has(name)) continue;
    seen.add(name);
    const use = info.bindings.get(name);
    if (!use) continue;
    for (const document of use.documents) documents.add(document);
    for (const local of use.locals) queue.push(local);
  }
  return documents;
}

export class ModuleGraph {
  constructor(paths) {
    this.paths = paths;
    this.modules = new Map();
    this.instanceClasses = new Map();
    this.methodsOf = new Map();
  }

  /** The service instances a module exports: `export const x = new X()`, by name. */
  instancesOf(file) {
    const cached = this.instanceClasses.get(file);
    if (cached) {
      return cached;
    }
    const instances = new Map();
    for (const statement of parseFile(file).program.body) {
      if (statement.type !== "ExportNamedDeclaration") continue;
      for (const declarator of statement.declaration?.declarations ?? []) {
        const init = declarator.init;
        if (
          declarator.id.type === "Identifier" &&
          init?.type === "NewExpression" &&
          init.callee.type === "Identifier"
        ) {
          instances.set(declarator.id.name, init.callee.name);
        }
      }
    }
    this.instanceClasses.set(file, instances);
    return instances;
  }

  /** The REST calls each method of the class `className` declared in `file` makes. */
  classMethods(file, className) {
    const key = `${file}#${className}`;
    const cached = this.methodsOf.get(key);
    if (cached) {
      return cached;
    }
    const methods = new Map();
    walk(parseFile(file).program, (node) => {
      if (node.type !== "ClassDeclaration" || node.id?.name !== className) return;
      for (const member of node.body.body) {
        if (member.type !== "MethodDefinition" && member.type !== "PropertyDefinition") {
          continue;
        }
        const name = member.key?.name;
        if (!name) continue;
        const calls = [];
        walk(member.value, (child) => {
          const rest = restCall(child);
          if (rest !== null) calls.push(rest);
        });
        methods.set(name, calls);
      }
    });
    this.methodsOf.set(key, methods);
    return methods;
  }

  /** What one module imports and calls. */
  module(file) {
    const cached = this.modules.get(file);
    if (cached) {
      return cached;
    }
    const { program } = parseFile(file);
    const info = {
      file,
      imports: [],
      dynamic: [],
      // The names this module imports from each module it imports, or ALL.
      names: new Map(),
      documents: new Set(),
      // Which documents each top-level binding uses, and which other
      // top-level bindings it reaches; `null` holds what the module's own
      // statements use whichever binding is imported.
      bindings: new Map(),
      exports: new Map(),
      rest: [],
      instanceCalls: [],
    };
    const instances = new Map();

    const addNames = (target, names) => {
      const current = info.names.get(target);
      if (current === ALL || names === ALL) {
        info.names.set(target, ALL);
        return;
      }
      const merged = current ?? new Set();
      for (const name of names) merged.add(name);
      info.names.set(target, merged);
    };
    const addEdge = (list, specifier) => {
      const target = resolveSpecifier(this.paths, specifier, file);
      if (target !== null) {
        list.push(target);
        addNames(target, ALL);
      }
    };

    for (const statement of program.body) {
      if (statement.type === "ImportDeclaration") {
        if (statement.importKind === "type") {
          continue;
        }
        const source = statement.source.value;
        const values = statement.specifiers.filter((spec) => spec.importKind !== "type");
        if (source === GRAPHQL_DOCUMENTS) {
          for (const spec of values) {
            const name = spec.imported?.name ?? spec.local.name;
            if (name.endsWith("Document")) {
              info.documents.add(name);
            }
          }
          continue;
        }
        const target = resolveSpecifier(this.paths, source, file);
        const exported = target === null ? new Map() : this.instancesOf(target);
        const followed = values.filter((spec) => {
          const className =
            spec.type === "ImportSpecifier" ? exported.get(spec.imported?.name) : undefined;
          if (className === undefined) return true;
          instances.set(spec.local.name, { file: target, className });
          return false;
        });
        if (
          target !== null &&
          (followed.length > 0 || (values.length === 0 && statement.specifiers.length === 0))
        ) {
          info.imports.push(target);
          const named = followed.every((spec) => spec.type === "ImportSpecifier");
          addNames(
            target,
            named && followed.length > 0 ? followed.map((spec) => spec.imported?.name) : ALL,
          );
        }
        continue;
      }
      if (
        (statement.type === "ExportNamedDeclaration" ||
          statement.type === "ExportAllDeclaration") &&
        statement.source &&
        statement.exportKind !== "type"
      ) {
        if (statement.source.value === GRAPHQL_DOCUMENTS) {
          for (const spec of statement.specifiers ?? []) {
            const name = spec.local?.name ?? spec.exported?.name;
            if (name?.endsWith("Document")) info.documents.add(name);
          }
          continue;
        }
        addEdge(info.imports, statement.source.value);
      }
    }
    if (info.documents.size > 0) {
      readBindings(program, info);
    }

    walk(program, (node) => {
      if (node.type === "ImportExpression" && node.source.type === "Literal") {
        const target = resolveSpecifier(this.paths, node.source.value, file);
        if (target !== null) {
          info.dynamic.push(target);
        }
        return;
      }
      const rest = restCall(node);
      if (rest !== null) {
        info.rest.push(rest);
        return;
      }
      const call = instanceCall(node, instances);
      if (call !== null) {
        info.instanceCalls.push(call);
      }
    });

    this.modules.set(file, info);
    return info;
  }

  /**
   * Every module reachable from `entries`, never entering a module in
   * `stop`. `follow(file, target, dynamic)` may refuse an edge.
   */
  closure(entries, { stop = new Set(), follow = () => true } = {}) {
    const seen = new Set();
    const queue = [...entries];
    while (queue.length > 0) {
      const file = queue.pop();
      if (seen.has(file)) continue;
      seen.add(file);
      const info = this.module(file);
      for (const target of info.imports) {
        if (!seen.has(target) && !stop.has(target) && follow(file, target, false)) {
          queue.push(target);
        }
      }
      for (const target of info.dynamic) {
        if (!seen.has(target) && !stop.has(target) && follow(file, target, true)) {
          queue.push(target);
        }
      }
    }

    return seen;
  }

  /**
   * What the modules in a closure take from each module that imports
   * documents: the names they import from it, or ALL when one loads it
   * whole (a dynamic import, a namespace import, a re-export) or it is an
   * entry.
   */
  wanted(files, entries) {
    const wanted = new Map(entries.map((entry) => [entry, ALL]));
    for (const file of files) {
      const info = this.module(file);
      for (const target of info.dynamic) wanted.set(target, ALL);
      for (const [target, names] of info.names) {
        const current = wanted.get(target);
        if (current === ALL) continue;
        if (names === ALL) {
          wanted.set(target, ALL);
          continue;
        }
        const merged = current ?? new Set();
        for (const name of names) merged.add(name);
        wanted.set(target, merged);
      }
    }
    return wanted;
  }

  /** The documents and REST calls a set of modules makes. */
  uses(files, entries) {
    const documents = new Set();
    const rest = new Map();
    const wanted = this.wanted(files, entries);
    const addRest = (call) => rest.set(`${call.method} ${call.path}`, call);
    for (const file of files) {
      const info = this.module(file);
      for (const name of documentsFor(info, wanted.get(file) ?? ALL)) documents.add(name);
      for (const call of info.rest) addRest(call);
      for (const { file: serviceFile, className, method } of info.instanceCalls) {
        for (const call of this.classMethods(serviceFile, className).get(method) ?? []) {
          addRest(call);
        }
      }
    }

    return { documents, rest: [...rest.values()] };
  }
}
