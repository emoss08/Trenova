import { IconProp } from "@/components/ui/icons";

/**
 * Options for search operations
 */
export interface SearchOptions {
  query: string;
  types?: string[];
  limit?: number;
  offset?: number;
  highlight?: boolean;
  facets?: string[];
  filter?: string;
}

/**
 * Individual search result document
 */
export interface SearchResult {
  id: string;
  type: string;
  title: string;
  description?: string;
  searchableText?: string;
  metadata?: Record<string, any>;
  highlights?: Record<string, string[]>;
  createdAt?: number;
  updatedAt?: number;
  businessUnitId?: string;
  organizationId?: string;
}

/**
 * Search response from the API
 */
export interface SearchResponse {
  results: SearchResult[];
  total: number;
  processedIn: string;
  query: string;
  facets?: Record<string, any>;
}

/**
 * Entity types for search
 */
export enum SearchEntityType {
  Shipment = "shipment",
  Driver = "driver",
  Equipment = "equipment",
  Customer = "customer",
  Location = "location",
  Route = "route",
  Order = "order",
  Invoice = "invoice",
}

/**
 * Search status constants
 */
export const SearchStatus = {
  // General status
  Active: "active",
  Inactive: "inactive",

  // Shipment status
  Planned: "planned",
  Dispatched: "dispatched",
  InTransit: "in_transit",
  Delivered: "delivered",
  Cancelled: "cancelled",
  Delayed: "delayed",

  // Equipment status
  Available: "available",
  InUse: "in_use",
  Maintenance: "maintenance",
  OutOfService: "out_of_service",

  // Driver status
  OnDuty: "on_duty",
  OffDuty: "off_duty",
  Driving: "driving",
  Rest: "rest",
  Vacation: "vacation",
  Sick: "sick",
  Training: "training",
} as const;

export type SearchStatusType = (typeof SearchStatus)[keyof typeof SearchStatus];

/**
 * Filter operators for search
 */
export const FilterOperator = {
  Equals: "=",
  NotEquals: "!=",
  GreaterThan: ">",
  LessThan: "<",
  GreaterThanOrEqual: ">=",
  LessThanOrEqual: "<=",
  In: "IN",
  NotIn: "NOT IN",
  Exists: "EXISTS",
  NotExists: "NOT EXISTS",
} as const;

export type FilterOperatorType =
  (typeof FilterOperator)[keyof typeof FilterOperator];

/**
 * Built-in sort options
 */
export const SortOption = {
  CreatedAtDesc: "createdAt:desc",
  CreatedAtAsc: "createdAt:asc",
  UpdatedAtDesc: "updatedAt:desc",
  UpdatedAtAsc: "updatedAt:asc",
  TitleAsc: "title:asc",
  TitleDesc: "title:desc",
} as const;

export type SortOptionType = (typeof SortOption)[keyof typeof SortOption];

export type SiteSearchTab =
  | "all"
  | "shipments"
  | "workers"
  | "equipment"
  | "customers";

export type SearchTabConfig = {
  icon: IconProp;
  label: string;
  color: string;
  filters: string[];
};

export type SearchInputProps = {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  activeTab: SiteSearchTab;
  setActiveTab: (tab: SiteSearchTab) => void;
  inputRef: React.RefObject<HTMLInputElement>;
  activeFilters?: Record<string, string>;
  setActiveFilters?: (filters: Record<string, string>) => void;
};
