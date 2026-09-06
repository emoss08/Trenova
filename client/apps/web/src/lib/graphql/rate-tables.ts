import {
  RateAgreementTableDocument,
  RateMatrixTableDocument,
  RateQuoteTableDocument,
  RateZoneTableDocument,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const rateAgreementTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RateAgreementTableDocument,
  operationName: "RateAgreementTable",
  connectionKey: "rateAgreements",
});

export type RateAgreementRow = DataTableConfigRow<typeof rateAgreementTableGraphQLConfig>;

export const rateZoneTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RateZoneTableDocument,
  operationName: "RateZoneTable",
  connectionKey: "rateZones",
});

export type RateZoneRow = DataTableConfigRow<typeof rateZoneTableGraphQLConfig>;

export const rateMatrixTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RateMatrixTableDocument,
  operationName: "RateMatrixTable",
  connectionKey: "rateMatrices",
});

export type RateMatrixRow = DataTableConfigRow<typeof rateMatrixTableGraphQLConfig>;

export const rateQuoteTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RateQuoteTableDocument,
  operationName: "RateQuoteTable",
  connectionKey: "rateQuotes",
});
