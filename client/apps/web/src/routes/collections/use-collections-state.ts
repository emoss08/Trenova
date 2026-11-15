import {
  searchBoolean,
  searchInteger,
  searchLiteral,
  searchString,
} from "@trenova/shared/lib/router/search-codec";

export const COLLECTOR_SCOPES = ["All", "Mine", "Unassigned"] as const;

export const collectionsSearchParamsParser = {
  customerId: searchString,
  scope: searchLiteral(COLLECTOR_SCOPES).withDefault("All"),
  due: searchBoolean.withDefault(false),
  query: searchString.withDefault(""),
  page: searchInteger.withDefault(1),
};
