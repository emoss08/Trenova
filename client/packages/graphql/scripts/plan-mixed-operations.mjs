// Finds client operations that select a root field of an add-on beside a root field of the
// base plan (or of another add-on). The server refuses an add-on's root field with
// PLAN_RESTRICTED when the organization's plan leaves the add-on out, and a refused non-null
// root fails the whole operation, so such an operation breaks its base screen on a plan
// without the add-on. The add-on root belongs in an operation of its own that the screen runs
// only when `usePlanFeature(feature).available`, or behind `@include`/`@skip` driven by it.
//
// A root field's feature is the feature of the resources it returns or checks: the kind of the
// type it returns (`@kind`, or a type named after a permission resource, or a connection of
// either), and every permission resource its Go resolver names. A root with neither counts
// as the base plan's. Each resource's feature and
// each feature's add-on are read from the Go source, which owns them.
import { readFileSync, readdirSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { Kind, parse } from "graphql";

const here = fileURLToPath(new URL(".", import.meta.url));
export const REPO_ROOT = resolve(here, "../../../..");
const TMS = join(REPO_ROOT, "services/tms/internal");
export const OPERATIONS_DIR = resolve(here, "../src/operations");
export const SCHEMA_PATH = resolve(here, "../src/schema.graphql");

export const BASE = "BASE";

function walk(dir, suffix) {
  const out = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(path, suffix));
    else if (entry.name.endsWith(suffix)) out.push(path);
  }
  return out.sort();
}

/** The camelCase-to-snake_case conversion the server uses to name a type's resource. */
export function camelToSnake(name) {
  let out = "";
  for (let i = 0; i < name.length; i += 1) {
    const ch = name[i];
    const upper = ch !== ch.toLowerCase();
    if (upper) {
      const prevLower = i > 0 && name[i - 1] !== name[i - 1].toUpperCase();
      const nextLower = i < name.length - 1 && name[i + 1] !== name[i + 1].toUpperCase();
      if (i > 0 && (prevLower || nextLower)) out += "_";
      out += ch.toLowerCase();
    } else {
      out += ch;
    }
  }
  return out;
}

/**
 * What the Go source says: each permission resource's feature, each feature's add-on (BASE
 * for the base plan) and the resources each root resolver checks.
 */
export function readGoCatalog() {
  const resourceNames = new Map();
  const resourceGen = readFileSync(
    join(TMS, "modules/identity/domain/permission/resource_gen.go"),
    "utf8",
  );
  for (const match of resourceGen.matchAll(/^\s*(Resource\w+)\s+Resource\s*=\s*"([^"]+)"/gm)) {
    resourceNames.set(match[1], match[2]);
  }

  const featureKeys = new Map();
  const constants = readFileSync(
    join(TMS, "modules/kernel/domain/platformcatalog/constants.go"),
    "utf8",
  );
  for (const match of constants.matchAll(/^\s*(Feature\w+)\s*=\s*FeatureKey\("([^"]+)"\)/gm)) {
    featureKeys.set(match[1], match[2]);
  }

  const addOnKeys = new Map();
  const addOns = readFileSync(join(TMS, "modules/kernel/domain/platformcatalog/addon.go"), "utf8");
  for (const match of addOns.matchAll(/^\s*(AddOn\w+)\s*=\s*AddOnKey\("([^"]+)"\)/gm)) {
    addOnKeys.set(match[1], match[2]);
  }

  const addOnOfFeature = new Map();
  const registry = readFileSync(
    join(TMS, "modules/kernel/domain/platformcatalog/registry.go"),
    "utf8",
  );
  for (const match of registry.matchAll(
    /Key:\s*(Feature\w+),\s*Name:\s*"[^"]*",\s*AddOn:\s*(AddOn\w+)/gs,
  )) {
    addOnOfFeature.set(featureKeys.get(match[1]), addOnKeys.get(match[2]));
  }

  const featureOfResource = new Map();
  const permsDir = join(TMS, "modules/identity/domain");
  for (const dir of readdirSync(permsDir).filter((name) => name.endsWith("perms"))) {
    for (const file of walk(join(permsDir, dir), ".go")) {
      if (file.endsWith("_test.go")) continue;
      let current = null;
      for (const line of readFileSync(file, "utf8").split("\n")) {
        const resource = /^\s*Resource:\s*permission\.(Resource\w+)\.String\(\)/.exec(line);
        if (resource) {
          current = resourceNames.get(resource[1]) ?? null;
          continue;
        }
        const feature = /^\s*Feature:\s*platformcatalog\.(Feature\w+)/.exec(line);
        if (feature && current && !featureOfResource.has(current)) {
          featureOfResource.set(current, featureKeys.get(feature[1]));
          current = null;
        }
      }
    }
  }

  const resolverChecks = new Map();
  for (const file of walk(join(TMS, "api/graphql/resolver"), ".go")) {
    if (file.endsWith("_test.go")) continue;
    const source = readFileSync(file, "utf8");
    for (const match of source.matchAll(/\nfunc \(r \*(Query|Mutation)Resolver\) (\w+)\(/g)) {
      const end = source.indexOf("\nfunc ", match.index + match[0].length);
      const body = source.slice(match.index + match[0].length, end === -1 ? undefined : end);
      const resources = new Set();
      for (const check of body.matchAll(/permission\.(Resource\w+)/g)) {
        const resource = resourceNames.get(check[1]);
        if (resource) resources.add(resource);
      }
      // gqlgen names the method with Go initialisms (WorkerDOTViolations for
      // workerDotViolations), so fields and methods are matched without case.
      resolverChecks.set(`${match[1].toLowerCase()}.${match[2].toLowerCase()}`, resources);
    }
  }

  return {
    resources: new Set(resourceNames.values()),
    featureOfResource,
    addOnOfFeature,
    resolverChecks,
  };
}

function namedType(type) {
  return type.kind === Kind.NAMED_TYPE ? type.name.value : namedType(type.type);
}

/** Each object type's fields (name → named type) and the kinds types declare with @kind. */
export function readSchema(sdl) {
  const fields = new Map();
  const kinds = new Map();
  for (const definition of parse(sdl).definitions) {
    if (
      definition.kind !== Kind.OBJECT_TYPE_DEFINITION &&
      definition.kind !== Kind.OBJECT_TYPE_EXTENSION
    ) {
      continue;
    }
    const name = definition.name.value;
    const own = fields.get(name) ?? new Map();
    for (const field of definition.fields ?? []) own.set(field.name.value, namedType(field.type));
    fields.set(name, own);
    const kind = definition.directives?.find((directive) => directive.name.value === "kind");
    const argument = kind?.arguments?.find((arg) => arg.name.value === "resource");
    if (argument?.value.kind === Kind.STRING) kinds.set(name, argument.value.value);
  }
  return { fields, kinds };
}

function kindsOfType(schema, catalog, typeName, seen = new Set()) {
  if (seen.has(typeName)) return [];
  seen.add(typeName);
  const declared = schema.kinds.get(typeName);
  if (declared) return [declared];
  const named = camelToSnake(typeName);
  if (schema.fields.has(typeName) && catalog.resources.has(named)) return [named];
  if (!typeName.endsWith("Connection") && !typeName.endsWith("Edge")) return [];
  return [...(schema.fields.get(typeName)?.values() ?? [])].flatMap((child) =>
    kindsOfType(schema, catalog, child, seen),
  );
}

/** The add-ons a root field belongs to (BASE for the base plan), with the features behind them. */
export function rootFieldPlan(schema, catalog, operationType, field) {
  const rootType = operationType === "mutation" ? "Mutation" : "Query";
  const returns = schema.fields.get(rootType)?.get(field);
  const resources = new Set([
    ...(returns ? kindsOfType(schema, catalog, returns) : []),
    ...(catalog.resolverChecks.get(`${operationType}.${field.toLowerCase()}`) ?? []),
  ]);
  const features = new Set();
  for (const resource of resources) {
    const feature = catalog.featureOfResource.get(resource);
    if (feature) features.add(feature);
  }
  const addOns = new Set(
    [...features].map((feature) => catalog.addOnOfFeature.get(feature) ?? BASE),
  );
  if (addOns.size === 0) addOns.add(BASE);
  return { features: [...features].sort(), addOns: [...addOns].sort() };
}

function hasCondition(node) {
  return (node.directives ?? []).some(
    (directive) => directive.name.value === "include" || directive.name.value === "skip",
  );
}

function rootSelections(selectionSet, fragments, conditional, out) {
  for (const selection of selectionSet.selections) {
    const guarded = conditional || hasCondition(selection);
    if (selection.kind === Kind.FIELD) {
      if (selection.name.value !== "__typename") {
        out.push({ field: selection.name.value, conditional: guarded });
      }
    } else if (selection.kind === Kind.INLINE_FRAGMENT) {
      rootSelections(selection.selectionSet, fragments, guarded, out);
    } else if (selection.kind === Kind.FRAGMENT_SPREAD) {
      const fragment = fragments.get(selection.name.value);
      if (fragment) rootSelections(fragment.selectionSet, fragments, guarded, out);
    }
  }
  return out;
}

/** Every query and mutation in the given `.graphql` sources, with its root fields. */
export function readOperations(files) {
  const fragments = new Map();
  const parsed = files.map(({ path, source }) => ({ path, document: parse(source) }));
  for (const { document } of parsed) {
    for (const definition of document.definitions) {
      if (definition.kind === Kind.FRAGMENT_DEFINITION) {
        fragments.set(definition.name.value, definition);
      }
    }
  }
  const operations = [];
  for (const { path, document } of parsed) {
    for (const definition of document.definitions) {
      if (definition.kind !== Kind.OPERATION_DEFINITION || !definition.name) continue;
      if (definition.operation === "subscription") continue;
      operations.push({
        path,
        name: definition.name.value,
        type: definition.operation,
        roots: rootSelections(definition.selectionSet, fragments, false, []),
      });
    }
  }
  return operations;
}

/**
 * The operations whose unconditional root fields span more than one add-on, or an add-on and
 * the base plan: on a plan without one of those add-ons the whole operation fails.
 */
export function mixedOperations(operations, schema, catalog) {
  const mixed = [];
  for (const operation of operations) {
    const roots = operation.roots.map((root) => ({
      ...root,
      ...rootFieldPlan(schema, catalog, operation.type, root.field),
    }));
    const groups = new Set(
      roots.filter((root) => !root.conditional).flatMap((root) => root.addOns),
    );
    const addOns = [...groups].filter((group) => group !== BASE);
    if (addOns.length > 0 && groups.size > 1) mixed.push({ ...operation, roots });
  }
  return mixed;
}

/** The repository's own operations, schema and catalog. */
export function loadRepository() {
  const files = walk(OPERATIONS_DIR, ".graphql").map((path) => ({
    path: relative(OPERATIONS_DIR, path),
    source: readFileSync(path, "utf8"),
  }));
  return {
    operations: readOperations(files),
    schema: readSchema(readFileSync(SCHEMA_PATH, "utf8")),
    catalog: readGoCatalog(),
  };
}
