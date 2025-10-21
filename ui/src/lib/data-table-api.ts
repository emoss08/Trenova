import type { FilterStateSchema } from "./schemas/table-configuration-schema";

export interface EnhancedAPIParams {
  limit?: number;
  offset?: number;
  query?: string;
  filters?: string;
  sort?: string;
  [key: string]: any;
}

export function convertFilterStateToAPIParams(
  filterState: FilterStateSchema,
  options: {
    additionalParams?: Record<string, any>;
  } = {},
): EnhancedAPIParams {
  const { additionalParams = {} } = options;

  const params: EnhancedAPIParams = {
    ...additionalParams,
  };

  if (filterState.globalSearch) {
    params.query = filterState.globalSearch;
  }

  if (filterState.filters.length > 0) {
    params.filters = JSON.stringify(filterState.filters);
  }

  if (filterState.sort.length > 0) {
    params.sort = JSON.stringify(filterState.sort);
  }

  return params;
}
