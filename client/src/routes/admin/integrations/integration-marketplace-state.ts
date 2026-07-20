import { parseAsString, parseAsStringLiteral } from "nuqs";

export const sortOrder = ["name_asc", "name_desc"];
export const statusOptions = ["all", "connected", "disconnected"];

export const integrationModalTypes = [
  "Samsara",
  "GoogleMaps",
  "OpenAI",
  "OpenWeatherMap",
  "OANDAExchangeRates",
  "EIAFuelPrices",
  "PCMiler",
  "Resend",
  "Postmark",
] as const;

export type IntegrationModalType = (typeof integrationModalTypes)[number];

export const searchParamsParser = {
  query: parseAsString.withDefault(""),
  sortBy: parseAsStringLiteral(sortOrder).withDefault("name_asc"),
  category: parseAsString.withDefault("all"),
  status: parseAsStringLiteral(statusOptions).withDefault("all"),
};

export const integrationHeaderSearchParamsParser = {
  query: searchParamsParser.query,
};

export const integrationCatalogSearchParamsParser = {
  sortBy: searchParamsParser.sortBy,
  category: searchParamsParser.category,
  status: searchParamsParser.status,
  query: searchParamsParser.query,
  type: parseAsStringLiteral(integrationModalTypes),
};
