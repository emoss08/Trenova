/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import type {
  EnhancedColumnDef,
  EnhancedQueryParams,
  FilterOperator,
  FilterUtils,
  URLFilterParams,
} from "@/types/enhanced-data-table";
import type {
  FilterStateSchema,
  SortFieldSchema,
} from "./schemas/table-configuration-schema";

/**
 * Utility functions for enhanced data table filtering and sorting
 */
export const filterUtils: FilterUtils = {
  /**
   * Serialize filter state to URL-safe parameters
   */
  serializeToURL(state: FilterStateSchema): URLFilterParams {
    const params: URLFilterParams = {};

    // Global search
    if (state.globalSearch) {
      params.query = state.globalSearch;
    }

    // Filters - always serialize, even if empty, to distinguish from "never set"
    params.filters =
      state.filters.length > 0 ? JSON.stringify(state.filters) : "[]";

    // Sort - always serialize, even if empty, to distinguish from "never set"
    params.sort = state.sort.length > 0 ? JSON.stringify(state.sort) : "[]";

    return params;
  },

  /**
   * Deserialize filter state from URL parameters
   */
  deserializeFromURL(params: URLFilterParams): FilterStateSchema {
    const state: FilterStateSchema = {
      filters: [],
      sort: [],
      globalSearch: "",
    };

    // Global search
    if (typeof params.query === "string") {
      state.globalSearch = params.query;
    }

    // Filters
    if (typeof params.filters === "string" && params.filters) {
      try {
        const filters = JSON.parse(params.filters);
        if (Array.isArray(filters)) {
          state.filters = filters;
        }
      } catch (error) {
        console.warn("Failed to parse filters from URL:", error);
      }
    }

    // Sort
    if (typeof params.sort === "string" && params.sort) {
      try {
        const sort = JSON.parse(params.sort);
        if (Array.isArray(sort)) {
          state.sort = sort;
        }
      } catch (error) {
        console.warn("Failed to parse sort from URL:", error);
      }
    }

    return state;
  },

  /**
   * Serialize filter state for API requests
   */
  serializeForAPI(state: FilterStateSchema): EnhancedQueryParams {
    const params: EnhancedQueryParams = {};

    if (state.globalSearch) {
      params.query = state.globalSearch;
    }

    if (state.filters.length > 0) {
      params.filters = state.filters;
    }

    if (state.sort.length > 0) {
      params.sort = state.sort;
    }

    return params;
  },

  /**
   * Validate filter state against column definitions
   */
  validateFilterState(
    state: FilterStateSchema,
    columns: EnhancedColumnDef<any>[],
  ): boolean {
    const filterableFields = new Set(
      columns
        .filter((col) => col.meta?.filterable)
        .map((col) => col.meta?.apiField || col.id)
        .filter(Boolean),
    );

    const sortableFields = new Set(
      columns
        .filter((col) => col.meta?.sortable)
        .map((col) => col.meta?.apiField || col.id)
        .filter(Boolean),
    );

    // Validate filters
    for (const filter of state.filters) {
      if (!filterableFields.has(filter.field)) {
        return false;
      }
    }

    // Validate sort fields
    for (const sort of state.sort) {
      if (!sortableFields.has(sort.field)) {
        return false;
      }
    }

    return true;
  },
};

/**
 * Convert filter state to query string format expected by backend
 */
export function serializeFiltersForBackend(
  filters: FilterStateSchema["filters"],
): Record<string, string> {
  const params: Record<string, string> = {};

  filters.forEach((filter, index) => {
    params[`filters[${index}][field]`] = filter.field;
    params[`filters[${index}][operator]`] = filter.operator;

    // Handle different value types
    if (Array.isArray(filter.value)) {
      params[`filters[${index}][value]`] = JSON.stringify(filter.value);
    } else if (typeof filter.value === "object" && filter.value !== null) {
      params[`filters[${index}][value]`] = JSON.stringify(filter.value);
    } else {
      params[`filters[${index}][value]`] = String(filter.value);
    }
  });

  return params;
}

/**
 * Convert sort state to query string format expected by backend
 */
export function serializeSortForBackend(
  sort: FilterStateSchema["sort"],
): Record<string, string> {
  const params: Record<string, string> = {};

  sort.forEach((sortField, index) => {
    params[`sort[${index}][field]`] = sortField.field;
    params[`sort[${index}][direction]`] = sortField.direction;
  });

  return params;
}

/**
 * Combine all parameters for backend API call
 */
export function buildAPIParams(
  filterState: FilterStateSchema,
  pagination: { page: number; pageSize: number },
): Record<string, string> {
  const params: Record<string, string> = {};

  // Pagination
  params.limit = String(pagination.pageSize);
  params.offset = String((pagination.page - 1) * pagination.pageSize);

  // Global search
  if (filterState.globalSearch) {
    params.query = filterState.globalSearch;
  }

  // Filters
  Object.assign(params, serializeFiltersForBackend(filterState.filters));

  // Sort
  Object.assign(params, serializeSortForBackend(filterState.sort));

  return params;
}

/**
 * Create a filter for a specific field and value
 */
export function createFieldFilter(
  field: string,
  operator: FilterStateSchema["filters"][number]["operator"],
  value: any,
): FilterStateSchema["filters"][number] {
  return { field, operator, value };
}

/**
 * Create a sort field
 */
export function createSortField(
  field: string,
  direction: FilterStateSchema["sort"][number]["direction"],
): FilterStateSchema["sort"][number] {
  return { field, direction };
}

/**
 * Default filter operators for different column types
 */
export const defaultFilterOperators = {
  text: "contains" as const,
  number: "eq" as const,
  date: "daterange" as const,
  boolean: "eq" as const,
  select: "eq" as const,
};

/**
 * Get available operators for a column type
 */
export function getAvailableOperators(filterType: string) {
  switch (filterType) {
    case "text":
      return [
        { value: "contains", label: "Contains" },
        { value: "startswith", label: "Starts with" },
        { value: "endswith", label: "Ends with" },
        { value: "eq", label: "Equals" },
        { value: "ne", label: "Not equals" },
      ];
    case "number":
      return [
        { value: "eq", label: "Equals" },
        { value: "ne", label: "Not equals" },
        { value: "gt", label: "Greater than" },
        { value: "gte", label: "Greater than or equal" },
        { value: "lt", label: "Less than" },
        { value: "lte", label: "Less than or equal" },
      ];
    case "date":
      return [
        { value: "daterange", label: "Date range" },
        { value: "eq", label: "Equals" },
        { value: "gt", label: "After" },
        { value: "lt", label: "Before" },
      ];
    case "select":
      return [
        { value: "eq", label: "Equals" },
        { value: "ne", label: "Not equals" },
        { value: "in", label: "In" },
        { value: "notin", label: "Not in" },
      ];
    case "boolean":
      return [
        { value: "eq", label: "Equals" },
        { value: "ne", label: "Not equals" },
      ];
    default:
      return [
        { value: "eq", label: "Equals" },
        { value: "ne", label: "Not equals" },
      ];
  }
}

export function getFilterOperatorLabel(operator: FilterOperator) {
  switch (operator) {
    case "eq":
      return "Equals";
    case "ne":
      return "Not equals";
    case "gt":
      return "Greater than";
    case "gte":
      return "Greater than or equal";
    case "lt":
      return "Less than";
    case "lte":
      return "Less than or equal";
    case "daterange":
      return "Date range";
    case "in":
      return "In";
    case "notin":
      return "Not in";
    case "contains":
      return "Contains";
    case "startswith":
      return "Starts with";
    case "endswith":
      return "Ends with";
    default:
      return operator;
  }
}

export function getSortDirectionLabel(direction: SortFieldSchema["direction"]) {
  switch (direction) {
    case "asc":
      return "Ascending";
    case "desc":
      return "Descending";
  }
}

/**
 * Debounced function creator for search input
 */
export function createDebouncedSearch(
  callback: (query: string) => void,
  delay: number = 300,
) {
  let timeoutId: ReturnType<typeof setTimeout>;

  return (query: string) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => callback(query), delay);
  };
}
