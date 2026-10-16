import {
  AuthEventsTableDocument,
  ExternalIdentitiesTableDocument,
  MfaAuthenticatorsTableDocument,
  RiskDecisionsTableDocument,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const authEventTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AuthEventsTableDocument,
  operationName: "AuthEventsTable",
  connectionKey: "authEvents",
});

export const riskDecisionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RiskDecisionsTableDocument,
  operationName: "RiskDecisionsTable",
  connectionKey: "riskDecisions",
});

export const externalIdentityTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ExternalIdentitiesTableDocument,
  operationName: "ExternalIdentitiesTable",
  connectionKey: "externalIdentities",
});

export const mfaAuthenticatorTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: MfaAuthenticatorsTableDocument,
  operationName: "MFAAuthenticatorsTable",
  connectionKey: "mfaAuthenticators",
});

export type AuthEventRow = DataTableConfigRow<typeof authEventTableGraphQLConfig>;
export type RiskDecisionRow = DataTableConfigRow<typeof riskDecisionTableGraphQLConfig>;
export type ExternalIdentityRow = DataTableConfigRow<typeof externalIdentityTableGraphQLConfig>;
export type MFAAuthenticatorRow = DataTableConfigRow<typeof mfaAuthenticatorTableGraphQLConfig>;
