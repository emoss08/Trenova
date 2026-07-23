import { createParser, parseAsBoolean, parseAsString } from "nuqs";

const parseAsStringArray = createParser<string[]>({
  parse: (value) => {
    if (!value) return [];
    try {
      return JSON.parse(value) as string[];
    } catch {
      return [];
    }
  },
  serialize: (value) => {
    if (!value || value.length === 0) return "";
    return JSON.stringify(value);
  },
}).withDefault([]);

export const queueSearchParamsParser = {
  item: parseAsString,
  status: parseAsString,
  query: parseAsString.withDefault(""),
  billType: parseAsString,
  billers: parseAsStringArray,
  includePosted: parseAsBoolean.withDefault(false),
  preset: parseAsString,
};

export const queueSelectionSearchParamsParser = {
  item: queueSearchParamsParser.item,
};

export const queueToolbarSearchParamsParser = {
  status: queueSearchParamsParser.status,
  includePosted: queueSearchParamsParser.includePosted,
};

export const queueSidebarSearchParamsParser = {
  status: queueSearchParamsParser.status,
  query: queueSearchParamsParser.query,
  billType: queueSearchParamsParser.billType,
  billers: queueSearchParamsParser.billers,
  includePosted: queueSearchParamsParser.includePosted,
  preset: queueSearchParamsParser.preset,
};
