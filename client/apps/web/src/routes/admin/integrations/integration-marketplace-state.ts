import { searchLiteral, searchString } from "@trenova/shared/lib/router/search-codec";

export const sortOrder = ["name_asc", "name_desc"];
export const statusOptions = ["all", "connected", "disconnected"];

export const integrationModalTypes = [
  "Samsara",
  "GoogleMaps",
  "OpenWeatherMap",
  "OANDAExchangeRates",
  "EIAFuelPrices",
  "PCMiler",
  "Resend",
  "Postmark",
  "WEXFuel",
  "ComdataFuel",
  "RampFuel",
  "CarrierOK",
  "FMCSAQCMobile",
  "QuickBooksOnline",
  "Xero",
  "BusinessCentral",
  "NetSuite",
] as const;

export const integrationSetupStates = ["connected"] as const;

export type IntegrationModalType = (typeof integrationModalTypes)[number];

export const searchParamsParser = {
  query: searchString.withDefault(""),
  sortBy: searchLiteral(sortOrder).withDefault("name_asc"),
  category: searchString.withDefault("all"),
  status: searchLiteral(statusOptions).withDefault("all"),
};

export const integrationHeaderSearchParamsParser = {
  query: searchParamsParser.query,
};

export const integrationCatalogSearchParamsParser = {
  sortBy: searchParamsParser.sortBy,
  category: searchParamsParser.category,
  status: searchParamsParser.status,
  query: searchParamsParser.query,
  type: searchLiteral(integrationModalTypes),
  setup: searchLiteral(integrationSetupStates),
};
