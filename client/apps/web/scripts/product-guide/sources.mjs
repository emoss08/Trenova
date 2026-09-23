/**
 * The app's own description of itself, read statically: which pages exist and
 * what guards them (the router), where they sit and what they are called (the
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
    router: join(src, "router.tsx"),
    navigation: join(src, "config", "navigation.config.ts"),
    recordLinks: join(src, "config", "record-links.ts"),
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

function joinPath(parent, child) {
  if (child.startsWith("/")) {
    return child;
  }

  return `${parent.replace(/\/$/, "")}/${child}`;
}

function callName(node) {
  return node?.type === "CallExpression" && node.callee.type === "Identifier"
    ? node.callee.name
    : null;
}

/** Every loader call in a route's `loader`, flattening combineLoaders(…). */
function loaderCalls(node) {
  if (!node) {
    return [];
  }
  if (callName(node) === "combineLoaders") {
    return node.arguments.flatMap(loaderCalls);
  }

  return [node];
}

function isRedirectLoader(node) {
  if (node?.type !== "ArrowFunctionExpression" && node?.type !== "FunctionExpression") {
    return false;
  }
  let redirects = false;
  walk(node.body, (child) => {
    if (callName(child) === "redirect") {
      redirects = true;
    }
  });

  return redirects;
}

/**
 * The module a lazy route loads, and the export it takes from it when it
 * destructures one (`const { FuelCardsPage } = await import(...)`). Two pages
 * can share a file; the export says which one this route renders.
 */
function lazyImport(lazyNode) {
  let specifier = null;
  let exportName = null;
  walk(lazyNode, (child) => {
    if (
      specifier === null &&
      child.type === "ImportExpression" &&
      child.source.type === "Literal"
    ) {
      specifier = child.source.value;
    }
    if (exportName !== null || child.type !== "VariableDeclarator") {
      return;
    }
    const init = child.init?.type === "AwaitExpression" ? child.init.argument : child.init;
    if (init?.type === "ImportExpression" && child.id.type === "ObjectPattern") {
      const first = child.id.properties.find((property) => property.type === "Property");
      if (first?.key?.type === "Identifier") {
        exportName = first.key.name;
      }
    }
  });

  return { specifier, exportName };
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
 * The pages a signed-in person can open: every route under a protected
 * loader that renders a page rather than redirecting, with the permissions
 * and capabilities its loaders (and its parents') demand.
 */
export function readRoutes(paths, members) {
  const { program } = parseFile(paths.router);
  const consts = topLevelConsts(program);
  const routesNode = consts.get("routes");
  if (routesNode?.type !== "ArrayExpression") {
    throw new Error(`${paths.router}: routes is not an array literal`);
  }

  const context = { members };
  const pages = new Map();

  const visit = (node, inherited) => {
    if (node?.type !== "ObjectExpression") {
      return;
    }
    const props = new Map();
    for (const property of node.properties) {
      if (property.type === "Property") {
        props.set(propertyName(property), property);
      }
    }

    const pathValue = props.has("path") ? evaluate(props.get("path").value, context) : undefined;
    const isIndex = props.has("index") && evaluate(props.get("index").value, context) === true;
    const path =
      typeof pathValue === "string" ? joinPath(inherited.path, pathValue) : inherited.path;

    const loader = props.get("loader")?.value;
    const redirect = isRedirectLoader(loader);
    const requires = [...inherited.requires];
    const capabilities = [...inherited.capabilities];
    let isProtected = inherited.isProtected;
    let isGuest = inherited.isGuest;

    for (const call of loaderCalls(loader)) {
      if (call?.type === "Identifier") {
        if (call.name === "protectedLoader") isProtected = true;
        if (call.name === "guestLoader") isGuest = true;
        continue;
      }
      const name = callName(call);
      if (name === "createPermissionLoader") {
        const resource = evaluate(call.arguments[0], context);
        const operation = call.arguments.length > 1 ? evaluate(call.arguments[1], context) : "read";
        if (resource !== UNKNOWN && operation !== UNKNOWN) {
          requires.push({ resource, operation });
        }
      } else if (name === "createCapabilityLoader") {
        const capability = evaluate(call.arguments[0], context);
        if (capability !== UNKNOWN) {
          capabilities.push(capability);
        }
      }
    }

    const lazy = props.get("lazy");
    const { specifier, exportName } = lazy
      ? lazyImport(lazy.value ?? lazy)
      : { specifier: null, exportName: null };
    const file = resolveModule(paths.src, specifier);

    if (
      (typeof pathValue === "string" || isIndex) &&
      !redirect &&
      file &&
      isProtected &&
      !isGuest
    ) {
      pages.set(path, {
        path,
        file,
        exportName,
        requires: dedupeRequirements(requires),
        capabilities: [...new Set(capabilities)],
        hasParams: path.includes(":"),
      });
    }

    const children = props.get("children")?.value;
    if (children?.type === "ArrayExpression") {
      for (const child of children.elements) {
        visit(child, { path, requires, capabilities, isProtected, isGuest });
      }
    }
  };

  for (const element of routesNode.elements) {
    visit(element, {
      path: "",
      requires: [],
      capabilities: [],
      isProtected: false,
      isGuest: false,
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
    if (candidate === file || /page\.tsx$/.test(candidate)) {
      continue;
    }
    const header = headerIn(candidate);
    if (header) {
      return header;
    }
  }

  return null;
}
