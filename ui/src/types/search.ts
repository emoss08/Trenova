import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";

export enum SearchEntityType {
  Shipment = "shipment",
  Customer = "customer",
  All = "all",
}

export type SearchRequest = {
  query: string;
  entityTypes: SearchEntityType[];
  limit?: number;
  offset?: number;
  filters?: Record<string, any>;
};

export type SearchOptions = {
  query: string;
  types?: string[];
  limit?: number;
  offset?: number;
  highlight?: boolean;
  facets?: string[];
  filter?: string;
};

export type SearchHit = {
  id: string;
  entityType: SearchEntityType;
  title: string;
  subtitle?: string;
  metadata?: Record<string, any>;
  score?: number;
  highlightedContent?: Record<string, string>;
};

export type SearchResponse = {
  hits: SearchHit[];
  total: number;
  offset: number;
  limit: number;
  processingTimeMs: number;
  query: string;
};

export type SearchInputProps = {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  activeTab: SearchEntityType;
  setActiveTab: (tab: SearchEntityType) => void;
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

export type SearchResultItemProps = {
  result: SearchHit;
  searchQuery: string;
};
