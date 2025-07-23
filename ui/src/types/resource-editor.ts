/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

// SchemaInformation holds all schema details for the requested schema.
export interface SchemaInformation {
  schemaName: string;
  tables: TableDetails[];
}

// TableDetails holds information about a specific table.
export interface TableDetails {
  tableName: string;
  columns: ColumnDetails[];
  indexes: IndexDetails[];
  constraints: ConstraintDetails[];
}

// ColumnDetails holds information about a table column.
export interface ColumnDetails {
  columnName: string;
  ordinalPosition: number;
  columnDefault?: string | null;
  isNullable: string; // "YES" or "NO"
  dataType: string;
  characterMaximumLength?: number | null;
  numericPrecision?: number | null;
  numericScale?: number | null;
  comment?: string | null;
}

// IndexDetails holds information about a table index.
export interface IndexDetails {
  indexName: string;
  indexDefinition: string; // SQL definition
  isUnique: boolean;
  isPrimary: boolean;
  indexType: string; // e.g., btree, hash
  columns: string[];
}

// ConstraintDetails holds information about a table constraint.
export interface ConstraintDetails {
  constraintName: string;
  constraintType: string; // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
  columnNames?: string[];
  foreignTableName?: string | null;
  foreignColumnNames?: string[];
  checkClause?: string | null;
  deferrable: boolean;
  initiallyDeferred: boolean;
}

// --- Autocomplete Types ---

// AutocompleteRequest defines the structure for autocomplete requests from frontend to backend.
export interface AutocompleteRequest {
  schemaName: string;
  tableName?: string; // Optional, if context is specific to a table
  currentQuery?: string; // The full query text so far
  prefix?: string; // The word/prefix the user is currently typing
}

// AutocompleteSuggestion defines a single suggestion item.
export interface AutocompleteSuggestion {
  value: string; // The actual text to be inserted
  caption: string; // Text displayed in the suggestion list (can be same as value)
  meta: string; // Type of suggestion (e.g., "table", "column", "keyword", "schema")
  score: number; // Score to rank suggestions (higher is better)
}

// AutocompleteResponse is the list of suggestions from the backend.
export interface AutocompleteResponse {
  suggestions: AutocompleteSuggestion[];
}

// --- SQL Query Execution Types (Frontend) ---

export interface ExecuteQueryRequest {
  schemaName: string;
  query: string;
  page?: number; // Current page number (1-indexed)
  pageSize?: number; // Number of rows per page
}

export interface QueryResult {
  columns: string[];
  rows: any[][]; // Each row is an array of any, matching backend's []any
  message?: string;
  error?: string;
  totalRows?: number;
  totalPages?: number;
  currentPage?: number;
  pageSize?: number;
}

export interface ExecuteQueryResponse {
  result: QueryResult;
}
