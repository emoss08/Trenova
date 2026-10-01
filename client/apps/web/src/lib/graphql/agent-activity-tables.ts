import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentExceptionTableDocument,
  AgentPlanTableDocument,
  AgentProposalTableDocument,
  AgentRunDetailDocument,
  AgentRunTableDocument,
  AgentRunTranscriptFieldsFragmentDoc,
  type AgentRunTranscriptFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const agentRunTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentRunTableDocument,
  operationName: "AgentRunTable",
  connectionKey: "agentRuns",
});

export type AgentRunRow = DataTableConfigRow<typeof agentRunTableGraphQLConfig>;

export type AgentRunTranscript = Omit<AgentRunTranscriptFieldsFragment, " $fragmentName">;
export type AgentRunTranscriptMessage = AgentRunTranscript["messages"][number];

/**
 * What a run's model said and the tools it called, or null for a run filed
 * before transcripts were kept, one that said nothing, or one that is not the
 * reader's organization's.
 */
export async function fetchAgentRunTranscript(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<AgentRunTranscript | null> {
  const data = await requestGraphQL({
    document: AgentRunDetailDocument,
    operationName: "AgentRunDetail",
    variables: { id },
    signal: options?.signal,
  });

  return getFragmentData(AgentRunTranscriptFieldsFragmentDoc, data.agentRun?.transcript) ?? null;
}

export const agentProposalTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentProposalTableDocument,
  operationName: "AgentProposalTable",
  connectionKey: "agentProposals",
});

export type AgentProposalRow = DataTableConfigRow<typeof agentProposalTableGraphQLConfig>;

export const agentPlanTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentPlanTableDocument,
  operationName: "AgentPlanTable",
  connectionKey: "agentPlans",
});

export type AgentPlanRow = DataTableConfigRow<typeof agentPlanTableGraphQLConfig>;

export const agentExceptionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentExceptionTableDocument,
  operationName: "AgentExceptionTable",
  connectionKey: "agentExceptions",
});

export type AgentExceptionRow = DataTableConfigRow<typeof agentExceptionTableGraphQLConfig>;
