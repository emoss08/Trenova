import type {
  FilterFieldSchema,
  FilterStateSchema,
  SortFieldSchema,
} from "@/lib/schemas/table-configuration-schema";
import type { DataTableProps } from "@/types/data-table";
import type { ColumnDef } from "@tanstack/react-table";
import type { SelectOption } from "./fields";

// Filter operators supported by the enhanced data table (must match backend)
export type FilterOperator =
  | "eq"
  | "ne"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "contains"
  | "startswith"
  | "endswith"
  | "like"
  | "ilike"
  | "in"
  | "notin"
  | "isnull"
  | "isnotnull"
  | "daterange";

// Enhanced query parameters that get sent to the backend
export interface EnhancedQueryParams {
  // Basic pagination
  limit?: number;
  offset?: number;
  query?: string;

  // Enhanced filtering and sorting
  filters?: FilterFieldSchema[];
  sort?: SortFieldSchema[];
}

// Column metadata for enhanced filtering
export interface EnhancedColumnMeta {
  // Field name for API requests
  apiField?: string;
  // Whether this column supports filtering
  filterable?: boolean;
  // Whether this column supports sorting
  sortable?: boolean;
  // Filter type for this column
  filterType?: "text" | "select" | "date" | "number" | "boolean";
  // Options for select filters
  filterOptions?: SelectOption[];
  // Default filter operator for this column
  defaultFilterOperator?: FilterOperator;
}

// Use standard ColumnDef with our extended meta interface
// The meta property is now properly typed through module augmentation
export type EnhancedColumnDef<TData, TValue = unknown> = ColumnDef<
  TData,
  TValue
>;

// Filter actions
export type FilterAction =
  | { type: "ADD_FILTER"; filter: FilterStateSchema["filters"] }
  | { type: "REMOVE_FILTER"; index: number }
  | {
      type: "UPDATE_FILTER";
      index: number;
      filter: FilterStateSchema["filters"];
    }
  | { type: "CLEAR_FILTERS" }
  | { type: "ADD_SORT"; sort: FilterStateSchema["sort"] }
  | { type: "REMOVE_SORT"; index: number }
  | { type: "UPDATE_SORT"; index: number; sort: FilterStateSchema["sort"] }
  | { type: "CLEAR_SORT" }
  | { type: "SET_GLOBAL_SEARCH"; query: string }
  | { type: "RESET_ALL" };

// Enhanced data table configuration
export interface EnhancedDataTableConfig {
  // Enable enhanced filtering
  enableFiltering?: boolean;
  // Enable enhanced sorting
  enableSorting?: boolean;
  // Enable multi-column sorting
  enableMultiSort?: boolean;
  // Maximum number of filters allowed
  maxFilters?: number;
  // Maximum number of sort fields allowed
  maxSorts?: number;
  // Debounce delay for search input (ms)
  searchDebounce?: number;
  // Show filter UI
  showFilterUI?: boolean;
  // Show sort UI
  showSortUI?: boolean;
  // Allow saving filter presets
  enableFilterPresets?: boolean;
}

// Filter preset for saving/loading common filter combinations
export interface FilterPreset {
  id: string;
  name: string;
  description?: string;
  filters: FilterStateSchema["filters"];
  sort: FilterStateSchema["sort"];
  globalSearch?: string;
  createdAt: Date;
  updatedAt: Date;
}

// URL serialization helpers
export interface URLFilterParams {
  [key: string]: string | string[] | undefined;
}

// Utility functions type definitions
export interface FilterUtils {
  // Serialize filters to URL-safe format
  serializeToURL(state: FilterStateSchema): URLFilterParams;
  // Deserialize filters from URL parameters
  deserializeFromURL(params: URLFilterParams): FilterStateSchema;
  // Serialize filters for API requests
  serializeForAPI(state: FilterStateSchema): EnhancedQueryParams;
  // Validate filter state
  validateFilterState(
    state: FilterStateSchema,
    columns: EnhancedColumnDef<any>[],
  ): boolean;
}

// Component prop types
export interface EnhancedDataTableProps<TData extends Record<string, any>>
  extends Omit<DataTableProps<TData>, "columns"> {
  columns: EnhancedColumnDef<TData>[];
  config?: EnhancedDataTableConfig;
  defaultFilters?: FilterFieldSchema[];
  defaultSort?: SortFieldSchema[];
  onFilterChange?: (state: FilterStateSchema) => void;
}

// Filter component props
export interface DataTableFilterProps {
  columns: EnhancedColumnDef<any>[];
  filterState: FilterStateSchema;
  onFilterChange: (state: FilterStateSchema) => void;
  config?: EnhancedDataTableConfig;
}

// Sort component props
export interface DataTableSortProps {
  columns: EnhancedColumnDef<any>[];
  sortState: FilterStateSchema["sort"];
  onSortChange: (sort: FilterStateSchema["sort"]) => void;
  config?: EnhancedDataTableConfig;
}

// Re-export existing types for convenience
export type { DataTableProps } from "@/types/data-table";

