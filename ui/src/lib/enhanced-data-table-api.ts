/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/**
 * Enhanced data table API utilities for converting frontend filter state to backend API calls
 */

import type { API_ENDPOINTS } from "@/types/server";
import type { FilterStateSchema } from "./schemas/table-configuration-schema";

export interface EnhancedAPIParams {
  // Base parameters
  limit?: number;
  offset?: number;
  query?: string;

  // Enhanced parameters (JSON encoded)
  filters?: string;
  sort?: string;

  // Additional parameters can be added by extending this interface
  [key: string]: any;
}

/**
 * Converts filter state to API parameters that support both legacy and enhanced backends
 */
export function convertFilterStateToAPIParams(
  filterState: FilterStateSchema,
  options: {
    useLegacyMode?: boolean;
    additionalParams?: Record<string, any>;
  } = {},
): EnhancedAPIParams {
  const { useLegacyMode = false, additionalParams = {} } = options;

  if (useLegacyMode) {
    // Legacy mode: Convert to simple query parameters
    return convertToLegacyParams(filterState, additionalParams);
  }

  // Enhanced mode: Use JSON encoding for complex filters
  const params: EnhancedAPIParams = {
    ...additionalParams,
  };

  // Add global search
  if (filterState.globalSearch) {
    params.query = filterState.globalSearch;
  }

  // Encode filters as JSON if present
  if (filterState.filters.length > 0) {
    params.filters = JSON.stringify(filterState.filters);
  }

  // Encode sort as JSON if present
  if (filterState.sort.length > 0) {
    params.sort = JSON.stringify(filterState.sort);
  }

  return params;
}

/**
 * Convert to legacy parameters for backward compatibility
 */
function convertToLegacyParams(
  filterState: FilterStateSchema,
  additionalParams: Record<string, any>,
): EnhancedAPIParams {
  const params: EnhancedAPIParams = {
    ...additionalParams,
  };

  // Add global search
  if (filterState.globalSearch) {
    params.query = filterState.globalSearch;
  }

  // Convert filters to simple key-value pairs (limited support)
  // Only handle eq operator for legacy mode
  filterState.filters.forEach((filter) => {
    if (filter.operator === "eq" && filter.value) {
      // Map field names if needed
      const paramName = mapFieldToParam(filter.field);
      params[paramName] = filter.value;
    }
    // Other operators are not supported in legacy mode
  });

  // Legacy doesn't support complex sorting, just use first sort
  if (filterState.sort.length > 0) {
    const firstSort = filterState.sort[0];
    params.sortBy = firstSort.field;
    params.sortOrder = firstSort.direction;
  }

  return params;
}

/**
 * Map field names to API parameter names
 */
function mapFieldToParam(field: string): string {
  const fieldMap: Record<string, string> = {
    customer_type: "customerType",
    created_at: "createdAt",
    updated_at: "updatedAt",
    // Add more mappings as needed
  };

  return fieldMap[field] || field;
}

/**
 * Build the API URL with enhanced parameters
 */
export function buildEnhancedAPIUrl(
  baseUrl: string,
  endpoint: API_ENDPOINTS | string,
  params: EnhancedAPIParams,
  pagination?: { page: number; pageSize: number },
): string {
  const url = new URL(`${baseUrl}${endpoint}`);

  // Add pagination
  if (pagination) {
    params.limit = pagination.pageSize;
    params.offset = (pagination.page - 1) * pagination.pageSize;
  }

  // Add all parameters to URL
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      url.searchParams.append(key, String(value));
    }
  });

  return url.toString();
}

/**
 * Check if the backend supports enhanced filtering
 * This can be determined by checking API version or feature flags
 */
export function supportsEnhancedFiltering(endpoint: string): boolean {
  // For now, check if endpoint has /enhanced suffix
  const enhancedEndpoints = [
    "/customers",
    "/shipments",
    "/workers",
    "/consolidations",
    "/email-profiles",
    "/hold-reasons",
  ];

  return enhancedEndpoints.some((e) => endpoint.includes(e));
}

/**
 * Get the appropriate endpoint based on feature support
 */
export function getDataTableEndpoint(
  resource: string,
  useEnhanced: boolean = false,
): API_ENDPOINTS {
  const endpoints: Record<
    string,
    { legacy: API_ENDPOINTS; enhanced: API_ENDPOINTS }
  > = {
    customer: {
      legacy: "/customers/",
      enhanced: "/customers/",
    },
    shipment: {
      legacy: "/shipments/",
      enhanced: "/shipments/",
    },
    audit_entry: {
      legacy: "/audit-logs/",
      enhanced: "/audit-logs/",
    },
    worker: {
      legacy: "/workers/",
      enhanced: "/workers/",
    },
    consolidation_group: {
      legacy: "/consolidations/",
      enhanced: "/consolidations/",
    },
    email_profile: {
      legacy: "/email-profiles/",
      enhanced: "/email-profiles/",
    },
    hold_reason: {
      legacy: "/hold-reasons/",
      enhanced: "/hold-reasons/",
    },
  };

  // Normalize resource name to lowercase
  const normalizedResource = resource.toLowerCase();
  const resourceEndpoints = endpoints[normalizedResource];

  if (!resourceEndpoints) {
    // Fallback to generic pattern
    return (
      useEnhanced ? `/${normalizedResource}s/` : `/${normalizedResource}s/`
    ) as API_ENDPOINTS;
  }

  return useEnhanced ? resourceEndpoints.enhanced : resourceEndpoints.legacy;
}
