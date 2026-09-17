import {
  AgentExceptionTableDocument,
  AgentProposalTableDocument,
  AgentRunTableDocument,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const agentRunTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentRunTableDocument,
  operationName: "AgentRunTable",
  connectionKey: "agentRuns",
});

export type AgentRunRow = DataTableConfigRow<typeof agentRunTableGraphQLConfig>;

export const agentProposalTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentProposalTableDocument,
  operationName: "AgentProposalTable",
  connectionKey: "agentProposals",
});

export type AgentProposalRow = DataTableConfigRow<typeof agentProposalTableGraphQLConfig>;

export const agentExceptionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentExceptionTableDocument,
  operationName: "AgentExceptionTable",
  connectionKey: "agentExceptions",
});

export type AgentExceptionRow = DataTableConfigRow<typeof agentExceptionTableGraphQLConfig>;
