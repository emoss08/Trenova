import catalogData from "@/graphql/generated/operation-catalog.json";
import type {
  CatalogFragment,
  CatalogOperation,
  CatalogSelection,
  CatalogVariable,
  OperationCatalog,
} from "@/types/graphql-catalog";

export const catalog = catalogData as OperationCatalog;

export const operationsByName = new Map(catalog.operations.map((op) => [op.name, op]));
export const fragmentsByName = new Map(catalog.fragments.map((fragment) => [fragment.name, fragment]));

export type CatalogFilter = "all" | "query" | "mutation" | "fragment";

export interface CatalogSearchResult {
  operations: CatalogOperation[];
  fragments: CatalogFragment[];
  total: number;
}

function scoreName(name: string, needle: string): number {
  const lower = name.toLowerCase();
  if (lower === needle) return 0;
  if (lower.startsWith(needle)) return 1;
  if (lower.includes(needle)) return 2;
  return 3;
}

function operationHaystack(op: CatalogOperation): string {
  return [op.name, op.domain, op.kind, op.sourceFile, ...op.rootFields, ...op.fragments]
    .join(" ")
    .toLowerCase();
}

function fragmentHaystack(fragment: CatalogFragment): string {
  return [fragment.name, fragment.typeCondition, fragment.domain, fragment.sourceFile]
    .join(" ")
    .toLowerCase();
}

export function searchCatalog(query: string, filter: CatalogFilter): CatalogSearchResult {
  const needle = query.trim().toLowerCase();
  const includeOps = filter === "all" || filter === "query" || filter === "mutation";
  const includeFragments = filter === "all" || filter === "fragment";

  let operations = includeOps ? catalog.operations : [];
  if (filter === "query") operations = operations.filter((op) => op.kind === "query");
  if (filter === "mutation") operations = operations.filter((op) => op.kind === "mutation");

  let fragments = includeFragments ? catalog.fragments : [];

  if (needle) {
    operations = operations
      .filter((op) => operationHaystack(op).includes(needle))
      .sort((a, b) => scoreName(a.name, needle) - scoreName(b.name, needle) || a.name.localeCompare(b.name));
    fragments = fragments
      .filter((fragment) => fragmentHaystack(fragment).includes(needle))
      .sort(
        (a, b) => scoreName(a.name, needle) - scoreName(b.name, needle) || a.name.localeCompare(b.name),
      );
  }

  return { operations, fragments, total: operations.length + fragments.length };
}

export function resolveSelection(selection: CatalogSelection | null): {
  operation: CatalogOperation | null;
  fragment: CatalogFragment | null;
} {
  if (!selection) {
    return { operation: null, fragment: null };
  }
  if (selection.kind === "operation") {
    return { operation: operationsByName.get(selection.name) ?? null, fragment: null };
  }
  return { operation: null, fragment: fragmentsByName.get(selection.name) ?? null };
}

const listTypePattern = /^\[(.+)\]$/;

function scaffoldValue(type: string): unknown {
  const required = type.endsWith("!");
  const inner = required ? type.slice(0, -1) : type;

  const listMatch = listTypePattern.exec(inner);
  if (listMatch) {
    return [];
  }

  switch (inner) {
    case "Int":
    case "Float":
      return 0;
    case "Boolean":
      return false;
    case "ID":
    case "String":
      return "";
    default:
      return null;
  }
}

export function scaffoldVariables(variables: CatalogVariable[]): string {
  if (variables.length === 0) {
    return "{}";
  }
  const scaffold: Record<string, unknown> = {};
  for (const variable of variables) {
    scaffold[variable.name] = scaffoldValue(variable.type);
  }
  return JSON.stringify(scaffold, null, 2);
}

export function parseSelectionParam(value: string | null): CatalogSelection | null {
  if (!value) {
    return null;
  }
  const [prefix, ...rest] = value.split(":");
  const name = rest.join(":");
  if (prefix === "fr" && fragmentsByName.has(name)) {
    return { kind: "fragment", name };
  }
  const operationName = prefix === "op" ? name : value;
  if (operationsByName.has(operationName)) {
    return { kind: "operation", name: operationName };
  }
  return null;
}

export function serializeSelectionParam(selection: CatalogSelection): string {
  return `${selection.kind === "fragment" ? "fr" : "op"}:${selection.name}`;
}
