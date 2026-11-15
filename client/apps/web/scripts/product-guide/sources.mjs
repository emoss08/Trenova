/**
 * The app's own description of itself, read statically: which pages exist and
 * what guards them (the route files), where they sit and what they are called (the
 * navigation config), what each says about itself (its page header), and
 * where records open (the record-link registry).
 */

import { existsSync, readdirSync, statSync } from "node:fs";
import { dirname, join, relative } from "node:path";

import { UNKNOWN, evaluate, known, parseFile, propertyName, topLevelConsts, walk } from "./ast.mjs";

export function sourcePaths(webRoot) {
  const src = join(webRoot, "src");
  const shared = join(webRoot, "..", "..", "packages", "shared", "src");

  return {
    src,
    pages: join(src, "pages"),
    navigation: join(src, "config", "navigation.config.ts"),
    recordLinks: join(src, "config", "record-links.ts"),
    recordKinds: join(shared, "types", "generated", "record-kinds.ts"),
    deskCards: join(src, "routes", "desk", "_components", "artifacts", "cards", "registry.tsx"),
    resources: join(shared, "types", "generated", "permission-resources.ts"),
    capabilities: join(shared, "types", "organization-capability.ts"),
  };
}

function objectConst(file, name) {
  const { program } = parseFile(file);
  const consts = topLevelConsts(program);
  const node = consts.get(name);
  if (!node) {
    throw new Error(`${file}: no top-level const ${name}`);
  }

  return known(evaluate(node, { scope: consts }));
}

/**
 * Member resolvers for `Resource.X`, `Operation.X` and
 * `OrganizationCapability.X`. Operations are named the way the server names
 * them (`read`, `create`), which is what the permission engine checks.
 */
export function loadMembers(paths) {
  const resources = objectConst(paths.resources, "Resource");
  const capabilities = objectConst(paths.capabilities, "OrganizationCapability");

  const lookup = (table, kind) => (member) => {
    if (!Object.hasOwn(table, member)) {
      throw new Error(`unknown ${kind} ${member}`);
    }
    return table[member];
  };

  return {
    Resource: lookup(resources, "Resource"),
    OrganizationCapability: lookup(capabilities, "OrganizationCapability"),
    Operation: (member) => member.charAt(0).toLowerCase() + member.slice(1),
  };
}

function callName(node) {
  return node?.type === "CallExpression" && node.callee.type === "Identifier"
    ? node.callee.name
    : null;
}

/** The route's options object: the argument of `createFileRoute("<id>")({ … })`. */
function fileRoute(program, file) {
  let found = null;
  walk(program, (node) => {
    if (
      found === null &&
      node.type === "CallExpression" &&
      callName(node.callee) === "createFileRoute" &&
      node.callee.arguments[0]?.type === "Literal"
    ) {
      found = { id: node.callee.arguments[0].value, options: node.arguments[0] };
    }
  });
  if (found === null || found.options?.type !== "ObjectExpression") {
    throw new Error(`${file}: no createFileRoute("<id>")({ … })`);
  }

  return found;
}

/** The module and export each imported name comes from. */
function importsOf(program) {
  const imports = new Map();
  for (const statement of program.body) {
    if (statement.type !== "ImportDeclaration") continue;
    for (const specifier of statement.specifiers) {
      const imported =
        specifier.type === "ImportSpecifier"
          ? (specifier.imported.name ?? specifier.imported.value)
          : null;
      imports.set(specifier.local.name, {
        specifier: statement.source.value,
        exportName: imported,
      });
    }
  }

  return imports;
}

const REDIRECT_GATES = new Set(["redirectTo", "redirectKeepingSearch"]);

/**
 * What a route's `beforeLoad: gates(…)` demands of everyone: each permission and capability
 * gate passed to it directly. A gate that only needs one of several (`anyOfGates`) demands
 * none of them on its own.
 */
function gateDemands(beforeLoad, context) {
  const demands = { requires: [], capabilities: [], redirects: false };
  if (callName(beforeLoad) !== "gates") {
    return demands;
  }
  for (const gate of beforeLoad.arguments) {
    const name = callName(gate);
    if (REDIRECT_GATES.has(name)) {
      demands.redirects = true;
    } else if (name === "requirePermission") {
      const resource = evaluate(gate.arguments[0], context);
      const operation = gate.arguments.length > 1 ? evaluate(gate.arguments[1], context) : "read";
      if (resource !== UNKNOWN && operation !== UNKNOWN) {
        demands.requires.push({ resource, operation });
      }
    } else if (name === "requireCapability") {
      const capability = evaluate(gate.arguments[0], context);
      if (capability !== UNKNOWN) {
        demands.capabilities.push(capability);
      }
    }
  }

  return demands;
}

function resolveModule(src, specifier) {
  if (!specifier?.startsWith("@/")) {
    return null;
  }
  const base = join(src, specifier.slice(2));
  for (const candidate of [`${base}.tsx`, `${base}.ts`, join(base, "index.tsx")]) {
    if (existsSync(candidate)) {
      return candidate;
    }
  }

  return null;
}

/**
 * The address a route id serves: pathless layouts (`_app`) and groups (`(public)`) dropped, the trailing `_` that
 * keeps a route out of its namesake's layout removed, an index's trailing slash trimmed.
 */
function pathOf(routeId) {
  const segments = routeId
    .split("/")
    .filter((segment) => segment !== "" && !segment.startsWith("_") && !segment.startsWith("("))
    .map((segment) => segment.replace(/_$/, ""));

  return `/${segments.join("/")}`;
}

const PROTECTED_LAYOUTS = ["/_app", "/_desk", "/_onboarding"];

function readRouteFiles(paths, members) {
  const context = { members };
  const routes = new Map();
  for (const entry of readdirSync(paths.pages, { recursive: true })) {
    const file = join(paths.pages, entry);
    if (!/\.tsx?$/.test(file) || entry.startsWith("__root")) continue;
    const { program } = parseFile(file);
    const { id, options } = fileRoute(program, file);
    const props = new Map();
    for (const property of options.properties) {
      if (property.type === "Property") {
        props.set(propertyName(property), property.value);
      }
    }
    const component = props.get("component");
    const page =
      component?.type === "Identifier" ? (importsOf(program).get(component.name) ?? null) : null;
    routes.set(id, {
      id,
      page,
      ...gateDemands(props.get("beforeLoad"), context),
    });
  }

  return routes;
}

/**
 * The pages a signed-in person can open: every route under a signed-in layout that renders a
 * page rather than redirecting, with the permissions and capabilities its gates (and its
 * layouts') demand. A route's layouts are the routes whose ids lead its own; one kept out of
 * its namesake's layout (`accounting_`) has an id no layout leads.
 */
export function readRoutes(paths, members) {
  const routes = readRouteFiles(paths, members);
  const pages = new Map();

  // The catch-all draws a section's index (the links under a module); a route that draws the
  // same page, like the admin area's own index, is that index too and not a page of its own.
  const sectionIndexes = new Set(
    [...routes.values()]
      .filter((route) => route.page !== null && route.id.endsWith("/$"))
      .map((route) => resolveModule(paths.src, route.page.specifier)),
  );

  // A route with children is a layout: its index, not the layout, is the page at its address.
  const ids = [...routes.keys()];
  const isLayout = (key) => ids.some((other) => other !== key && other.startsWith(`${key}/`));

  for (const [key, route] of routes) {
    if (route.redirects || route.page === null || isLayout(key)) continue;
    if (!PROTECTED_LAYOUTS.some((layout) => key === layout || key.startsWith(`${layout}/`))) {
      continue;
    }
    // An index (`/_desk/desk/`) stands in the layout whose id is its own without the slash.
    const segments = key.replace(/(.)\/$/, "$1").split("/");
    const requires = [];
    const capabilities = [];
    for (let length = 2; length <= segments.length; length++) {
      const at = segments.slice(0, length).join("/");
      const layout = at === key ? undefined : routes.get(at);
      if (layout) {
        requires.push(...layout.requires);
        capabilities.push(...layout.capabilities);
      }
    }
    requires.push(...route.requires);
    capabilities.push(...route.capabilities);
    const file = resolveModule(paths.src, route.page.specifier);
    const path = pathOf(route.id);
    if (!file || path.endsWith("/$") || sectionIndexes.has(file)) continue;
    pages.set(path, {
      path,
      file,
      exportName: route.page.exportName,
      requires: dedupeRequirements(requires),
      capabilities: [...new Set(capabilities)],
      hasParams: path.includes("$"),
    });
  }

  return pages;
}

function dedupeRequirements(requires) {
  const seen = new Map();
  for (const requirement of requires) {
    seen.set(`${requirement.resource}:${requirement.operation}`, requirement);
  }

  return [...seen.values()];
}

/** Modules, their pages, the quick actions and the admin links. */
export function readNavigation(paths, members) {
  const { program } = parseFile(paths.navigation);
  const consts = topLevelConsts(program);
  const context = { scope: consts, members };

  const config = known(evaluate(consts.get("navigationConfig"), context));
  const adminLinks = known(evaluate(consts.get("adminLinks"), context));

  return {
    modules: config.modules ?? [],
    quickActions: config.quickActions ?? [],
    adminLinks: adminLinks ?? [],
  };
}

export function readRecordLinks(paths) {
  return objectConst(paths.recordLinks, "RECORD_LINKS");
}

/** The kinds of record the server knows, generated from its permission registry. */
export function readRecordKinds(paths) {
  return new Set(Object.keys(objectConst(paths.recordKinds, "RecordKindIDPrefix")));
}

/**
 * The kinds of record the Desk draws as a card of its own: the keys of the
 * card registry. Its values are lazily loaded components, which a static read
 * cannot evaluate and does not need.
 */
export function readDeskCards(paths) {
  const { program } = parseFile(paths.deskCards);
  let node = topLevelConsts(program).get("RECORD_CARDS");
  while (node && node.type !== "ObjectExpression" && node.expression) {
    node = node.expression;
  }
  if (node?.type !== "ObjectExpression") {
    throw new Error(`${paths.deskCards}: no top-level object RECORD_CARDS`);
  }

  return new Set(node.properties.map(propertyName).filter((name) => name !== null));
}

function jsxAttributeObject(element, name) {
  for (const attribute of element.attributes ?? []) {
    if (
      attribute.type === "JSXAttribute" &&
      attribute.name?.name === name &&
      attribute.value?.type === "JSXExpressionContainer"
    ) {
      return attribute.value.expression;
    }
  }

  return null;
}

/**
 * The top-level declaration of one export, so a file holding two pages is
 * read one page at a time. The whole file when the export is not found.
 */
function exportScope(program, exportName) {
  if (!exportName) {
    return program;
  }
  for (const statement of program.body) {
    const declaration =
      statement.type === "ExportNamedDeclaration" ? statement.declaration : statement;
    if (declaration?.type === "FunctionDeclaration" && declaration.id?.name === exportName) {
      return declaration;
    }
    if (declaration?.type === "VariableDeclaration") {
      const declarator = declaration.declarations.find(
        (candidate) => candidate.id.type === "Identifier" && candidate.id.name === exportName,
      );
      if (declarator) {
        return declarator;
      }
    }
  }

  return program;
}

function headerIn(file, exportName = null) {
  const { program } = parseFile(file);
  let header = null;
  walk(exportScope(program, exportName), (node) => {
    if (header !== null || node.type !== "JSXOpeningElement") {
      return;
    }
    const props = jsxAttributeObject(node, "pageHeaderProps");
    if (props?.type !== "ObjectExpression") {
      return;
    }
    const value = known(evaluate(props, {}));
    if (typeof value.title === "string") {
      header = {
        title: value.title,
        description: typeof value.description === "string" ? value.description : "",
      };
    }
  });

  return header;
}

function routeFiles(directory) {
  const files = [];
  for (const entry of readdirSync(directory)) {
    if (entry === "__tests__" || entry.startsWith(".")) {
      continue;
    }
    const full = join(directory, entry);
    if (statSync(full).isDirectory()) {
      files.push(...routeFiles(full));
    } else if (
      entry.endsWith(".tsx") &&
      !entry.includes(".test.") &&
      !entry.includes(".stories.")
    ) {
      files.push(full);
    }
  }

  return files;
}

/**
 * What a page says about itself in its header. The page file usually renders
 * `PageLayout` itself; when it delegates, the header is in its own route
 * directory, never in a sibling page's.
 */
export function readPageHeader(file, exportName = null) {
  const own = headerIn(file, exportName);
  if (own) {
    return own;
  }

  const directory = dirname(file);
  if (!relative(directory, file).startsWith("page")) {
    return null;
  }
  for (const candidate of routeFiles(directory).sort()) {
    // Another page in the same directory is a different page with its own
    // header; only the components this page is built from speak for it.
    if (candidate === file || candidate.endsWith("page.tsx")) {
      continue;
    }
    const header = headerIn(candidate);
    if (header) {
      return header;
    }
  }

  return null;
}
