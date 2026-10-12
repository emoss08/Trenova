import {
  type FieldFilter,
  type FilterGroup,
  type SortField,
} from "@trenova/shared/types/data-table";
import { createParser, parseAsInteger, parseAsString, parseAsStringLiteral } from "nuqs";

export const parseAsSortFields = createParser<SortField[]>({
  parse: (value) => {
    if (!value) return [];
    try {
      return JSON.parse(value) as SortField[];
    } catch {
      return [];
    }
  },
  serialize: (value) => {
    if (!value || value.length === 0) return "";
    return JSON.stringify(value);
  },
}).withDefault([]);

export const parseAsFilterGroups = createParser<FilterGroup[]>({
  parse: (value) => {
    if (!value) return [];
    try {
      return JSON.parse(value) as FilterGroup[];
    } catch {
      return [];
    }
  },
  serialize: (value) => {
    if (!value || value.length === 0) return "";
    return JSON.stringify(value);
  },
}).withDefault([]);

export const parseAsFieldFilters = createParser<FieldFilter[]>({
  parse: (value) => {
    if (!value) return [];
    try {
      return JSON.parse(value) as FieldFilter[];
    } catch {
      return [];
    }
  },
  serialize: (value) => {
    if (!value || value.length === 0) return "";
    return JSON.stringify(value);
  },
}).withDefault([]);

export const entitySearchParamsParser = {
  entityId: parseAsString,
  modalType: parseAsStringLiteral(["edit", "create"]),
};

export const panelSearchParamsParser = {
  panelType: parseAsStringLiteral(["edit", "create"]),
  panelEntityId: parseAsString,
};

export const DEFAULT_PAGE_SIZE = 10;

export const tablePaginationSearchParamsParser = {
  pageIndex: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(DEFAULT_PAGE_SIZE),
};

/**
 * The page size a table actually runs at. A table that names its sizes opens
 * at the first of them, and a size it does not offer (an old link, a saved
 * view from before the sizes changed) falls back to that first size rather
 * than fetching a page the picker cannot show.
 */
export function resolvePageSize(pageSize: number, pageSizeOptions?: readonly number[]): number {
  if (!pageSizeOptions?.length || pageSizeOptions.includes(pageSize)) return pageSize;
  return pageSizeOptions[0];
}

export function defaultPageSizeFor(pageSizeOptions?: readonly number[]): number {
  return pageSizeOptions?.[0] ?? DEFAULT_PAGE_SIZE;
}

export const tableFilterSearchParamsParser = {
  query: parseAsString.withDefault(""),
  fieldFilters: parseAsFieldFilters,
  filterGroups: parseAsFilterGroups,
  sort: parseAsSortFields,
};

export const searchParamsParser = {
  ...entitySearchParamsParser,
  ...panelSearchParamsParser,
  ...tablePaginationSearchParamsParser,
  ...tableFilterSearchParamsParser,
};
