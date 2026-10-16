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

/** The sizes a table offers when it names none; the server serves at most 100 rows a page. */
export const DEFAULT_PAGE_SIZE_OPTIONS: readonly number[] = [25, 50, 100];
export const DEFAULT_PAGE_SIZE = 50;

export const tablePaginationSearchParamsParser = {
  pageIndex: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(DEFAULT_PAGE_SIZE),
};

/**
 * The page size a table actually runs at. A table that names its sizes opens
 * at the first of them, one that does not opens at the default size, and a
 * size it does not offer (an old link, a saved view from before the sizes
 * changed) falls back to where it opens rather than fetching a page the picker
 * cannot show.
 */
export function resolvePageSize(pageSize: number, pageSizeOptions?: readonly number[]): number {
  const offered = pageSizeOptions?.length ? pageSizeOptions : DEFAULT_PAGE_SIZE_OPTIONS;
  return offered.includes(pageSize) ? pageSize : defaultPageSizeFor(pageSizeOptions);
}

export function defaultPageSizeFor(pageSizeOptions?: readonly number[]): number {
  return pageSizeOptions?.length ? pageSizeOptions[0] : DEFAULT_PAGE_SIZE;
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
