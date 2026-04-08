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
