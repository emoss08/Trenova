import {
  type FieldFilter,
  type FilterGroup,
  type SortField,
} from "@/types/data-table";
import {
  createParser,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
} from "nuqs";

const parseAsSortFields = createParser<SortField[]>({
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

const parseAsFilterGroups = createParser<FilterGroup[]>({
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

const parseAsFieldFilters = createParser<FieldFilter[]>({
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

export const searchParamsParser = {
  entityId: parseAsString,
  modalType: parseAsStringLiteral(["edit", "create"]),
  panelType: parseAsStringLiteral(["edit", "create"]),
  panelEntityId: parseAsString,
  pageIndex: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(10),
  query: parseAsString.withDefault(""),
  fieldFilters: parseAsFieldFilters,
  filterGroups: parseAsFilterGroups,
  sort: parseAsSortFields,
};
