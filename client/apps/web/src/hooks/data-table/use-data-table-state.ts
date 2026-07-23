import {
  type FieldFilter,
  type FilterGroup,
  type SortField,
} from "@trenova/shared/types/data-table";
import {
  createParser,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
} from "nuqs";

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

export const tablePaginationSearchParamsParser = {
  pageIndex: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(10),
};

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
