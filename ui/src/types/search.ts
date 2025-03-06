import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";

export type SiteSearchTab =
  | "all"
  | "shipments"
  | "workers"
  | "tractors"
  | "customers";

/**
 * Entity types for search
 */
export enum SearchEntityType {
  Shipments = "shipments",
  Workers = "workers",
  Tractors = "tractors",
  Customers = "customers",
}

/**
 * Options for search operations
 */
export type SearchOptions = {
  query: string;
  types?: string[];
  limit?: number;
  offset?: number;
  highlight?: boolean;
  facets?: string[];
  filter?: string;
};

/**
 * Individual search result document
 */
export type SearchResult = {
  id: string;
  type: SearchEntityType;
  title: string;
  description?: string;
  searchableText?: string;
  metadata?: Record<string, any>;
  highlights?: Record<string, string[]>;
  createdAt?: number;
  updatedAt?: number;
  businessUnitId?: string;
  organizationId?: string;
};

/**
 * Search response from the API
 */
export type SearchResponse = {
  results: SearchResult[];
  total: number;
  processedIn: string;
  query: string;
  facets?: Record<string, any>;
};

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

export type SearchInputProps = {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  activeTab: SiteSearchTab;
  setActiveTab: (tab: SiteSearchTab) => void;
  inputRef: React.RefObject<HTMLInputElement | null>;
  activeFilters?: Record<string, string>;
  setActiveFilters?: (filters: Record<string, string>) => void;
};

export type SiteSearchQuickOptionProps = {
  icon: IconDefinition;
  label: string;
  description: string;
  link?: string;
  onClick?: () => void;
};
