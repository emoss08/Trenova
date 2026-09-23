import {
  AgentEvalCaseDetailDocument,
  AgentEvalCaseDetailFieldsFragmentDoc,
  AgentEvalCaseTableDocument,
  AgentEvalCaseTableRowFieldsFragmentDoc,
  CreateAgentEvalCaseDocument,
  ReplayAgentEvalCaseDocument,
  SetAgentEvalCaseStatusDocument,
  UpdateAgentEvalCaseDocument,
  type AgentEvalCaseDetailFieldsFragment,
  type AgentEvalCaseStatus,
  type AgentEvalCaseTableRowFieldsFragment,
  type CreateAgentEvalCaseInput,
  type UpdateAgentEvalCaseInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const agentEvalCaseTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentEvalCaseTableDocument,
  operationName: "AgentEvalCaseTable",
  connectionKey: "agentEvalCases",
});

export type AgentEvalCaseRow = DataTableConfigRow<typeof agentEvalCaseTableGraphQLConfig>;

export type AgentEvalCaseDetail = Omit<AgentEvalCaseDetailFieldsFragment, " $fragmentRefs"> &
  Omit<AgentEvalCaseTableRowFieldsFragment, " $fragmentName">;

export const AGENT_EVAL_CASE_LIST_KEY = "agent-eval-case-list";
export const AGENT_EVAL_CASE_DETAIL_KEY = "agent-eval-case";

function unmaskCase(
  masked: FragmentType<typeof AgentEvalCaseDetailFieldsFragmentDoc>,
): AgentEvalCaseDetail {
  const detail = getFragmentData(AgentEvalCaseDetailFieldsFragmentDoc, masked);
  const row = getFragmentData(AgentEvalCaseTableRowFieldsFragmentDoc, detail);

  return { ...detail, ...row };
}

export async function fetchAgentEvalCase(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<AgentEvalCaseDetail | null> {
  const data = await requestGraphQL({
    document: AgentEvalCaseDetailDocument,
    operationName: "AgentEvalCaseDetail",
    variables: { id },
    signal: options?.signal,
  });

  return data.agentEvalCase ? unmaskCase(data.agentEvalCase) : null;
}

export async function createAgentEvalCase(
  input: CreateAgentEvalCaseInput,
): Promise<{ evalCase: AgentEvalCaseDetail; duplicate: boolean }> {
  const data = await requestGraphQL({
    document: CreateAgentEvalCaseDocument,
    operationName: "CreateAgentEvalCase",
    variables: { input },
  });

  return {
    evalCase: unmaskCase(data.createAgentEvalCase.evalCase),
    duplicate: data.createAgentEvalCase.duplicate,
  };
}

export async function updateAgentEvalCase(
  id: string,
  input: UpdateAgentEvalCaseInput,
): Promise<AgentEvalCaseDetail> {
  const data = await requestGraphQL({
    document: UpdateAgentEvalCaseDocument,
    operationName: "UpdateAgentEvalCase",
    variables: { id, input },
  });

  return unmaskCase(data.updateAgentEvalCase);
}

export async function setAgentEvalCaseStatus(
  id: string,
  status: AgentEvalCaseStatus,
): Promise<AgentEvalCaseDetail> {
  const data = await requestGraphQL({
    document: SetAgentEvalCaseStatusDocument,
    operationName: "SetAgentEvalCaseStatus",
    variables: { id, status },
  });

  return unmaskCase(data.setAgentEvalCaseStatus);
}

export async function replayAgentEvalCase(id: string) {
  const data = await requestGraphQL({
    document: ReplayAgentEvalCaseDocument,
    operationName: "ReplayAgentEvalCase",
    variables: { id },
  });

  return data.replayAgentEvalCase;
}
