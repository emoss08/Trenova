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

export interface OperationCatalog {
  operationCount: number;
  fragmentCount: number;
  operations: CatalogOperation[];
  fragments: CatalogFragment[];
}

export type CatalogSelectionKind = "operation" | "fragment";

export interface CatalogSelection {
  kind: CatalogSelectionKind;
  name: string;
}
