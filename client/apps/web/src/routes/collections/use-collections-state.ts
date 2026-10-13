import { parseAsBoolean, parseAsInteger, parseAsString, parseAsStringLiteral } from "nuqs";

export const COLLECTOR_SCOPES = ["All", "Mine", "Unassigned"] as const;

export const collectionsSearchParamsParser = {
  customerId: parseAsString,
  scope: parseAsStringLiteral(COLLECTOR_SCOPES).withDefault("All"),
  due: parseAsBoolean.withDefault(false),
  query: parseAsString.withDefault(""),
  page: parseAsInteger.withDefault(1),
};
