import {
  RateAgreementTableDocument,
  RateMatrixTableDocument,
  RateQuoteTableDocument,
  RateZoneTableDocument,
  type RateAgreementTableQueryVariables,
  type RateMatrixTableQueryVariables,
  type RateQuoteTableQueryVariables,
  type RateZoneTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { RateAgreement, RateMatrix, RateQuote, RateZone } from "@trenova/shared/types/rate";

export const rateAgreementTableGraphQLConfig = defineDataTableGraphQLConfig<
  RateAgreement,
  RateAgreementTableQueryVariables
>({
  document: RateAgreementTableDocument,
  operationName: "RateAgreementTable",
  connectionKey: "rateAgreements",
});

export const rateZoneTableGraphQLConfig = defineDataTableGraphQLConfig<
  RateZone,
  RateZoneTableQueryVariables
>({
  document: RateZoneTableDocument,
  operationName: "RateZoneTable",
  connectionKey: "rateZones",
});

export const rateMatrixTableGraphQLConfig = defineDataTableGraphQLConfig<
  RateMatrix,
  RateMatrixTableQueryVariables
>({
  document: RateMatrixTableDocument,
  operationName: "RateMatrixTable",
  connectionKey: "rateMatrices",
});

export const rateQuoteTableGraphQLConfig = defineDataTableGraphQLConfig<
  RateQuote,
  RateQuoteTableQueryVariables
>({
  document: RateQuoteTableDocument,
  operationName: "RateQuoteTable",
  connectionKey: "rateQuotes",
});
