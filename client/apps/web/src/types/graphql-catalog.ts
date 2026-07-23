export type CatalogOperationKind = "query" | "mutation" | "subscription";

export interface CatalogVariable {
  name: string;
  type: string;
  defaultValue: string | null;
}

export interface CatalogOperation {
  name: string;
  kind: CatalogOperationKind;
  domain: string;
  sourceFile: string;
  hash: string | null;
  rootFields: string[];
  variables: CatalogVariable[];
  fragments: string[];
  usages: string[];
  sdl: string;
}

export interface CatalogFragment {
  name: string;
  typeCondition: string;
  domain: string;
  sourceFile: string;
  fragments: string[];
  usedByOperations: string[];
  usages: string[];
  sdl: string;
}

export interface CatalogInputField {
  name: string;
  type: string;
  defaultValue: string | null;
  defaultJson?: unknown;
}

export interface CatalogInputType {
  kind: "input";
  fields: CatalogInputField[];
  sdl: string;
}

export interface CatalogEnumType {
  kind: "enum";
  values: string[];
  sdl: string;
}

export interface CatalogScalarType {
  kind: "scalar";
  sdl: string;
}

export type CatalogNamedType = CatalogInputType | CatalogEnumType | CatalogScalarType;

export interface OperationCatalog {
  operationCount: number;
  fragmentCount: number;
  operations: CatalogOperation[];
  fragments: CatalogFragment[];
  types: Record<string, CatalogNamedType>;
}

export type CatalogSelectionKind = "operation" | "fragment";

export interface CatalogSelection {
  kind: CatalogSelectionKind;
  name: string;
}
