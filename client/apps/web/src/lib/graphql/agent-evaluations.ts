import {
  AgentEvaluationDetailDocument,
  AgentEvaluationTableDocument,
  ReplayAgentRunDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const agentEvaluationTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentEvaluationTableDocument,
  operationName: "AgentEvaluationTable",
  connectionKey: "agentEvaluations",
});

export type AgentEvaluationRow = DataTableConfigRow<typeof agentEvaluationTableGraphQLConfig>;

export const AGENT_EVALUATION_LIST_KEY = "agent-evaluation-list";

export async function fetchAgentEvaluation(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: AgentEvaluationDetailDocument,
    operationName: "AgentEvaluationDetail",
    variables: { id },
    signal: options?.signal,
  });

  return data.agentEvaluation;
}

export type AgentEvaluationDetail = NonNullable<Awaited<ReturnType<typeof fetchAgentEvaluation>>>;

export async function replayAgentRun(runId: string) {
  const data = await requestGraphQL({
    document: ReplayAgentRunDocument,
    operationName: "ReplayAgentRun",
    variables: { runId },
  });

  return data.replayAgentRun;
}
