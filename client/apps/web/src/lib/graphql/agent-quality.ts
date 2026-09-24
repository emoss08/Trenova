import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentQualityAgentTableDocument,
  AgentQualityControlDocument,
  AgentQualityControlFieldsFragmentDoc,
  AgentQualityDocument,
  AgentQualityOverviewDocument,
  AgentQualityPointFieldsFragmentDoc,
  AgentSuiteRunCaseTableDocument,
  AgentSuiteRunDocument,
  AgentSuiteRunFieldsFragmentDoc,
  AgentSuiteRunTableDocument,
  AgentWorstRatedAnswerFieldsFragmentDoc,
  AgentWorstRatedAnswerTableDocument,
  RunAgentSuiteDocument,
  UpdateAgentQualityControlDocument,
  type AgentQualityControlFieldsFragment,
  type AgentQualityOverviewQuery,
  type AgentQualityPointFieldsFragment,
  type AgentSuiteRunCaseFieldsFragment,
  type AgentSuiteRunFieldsFragment,
  type AgentSuiteRunStatus,
  type AgentWorstRatedAnswerFieldsFragment,
  type UpdateAgentQualityControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export type AgentSuiteRun = AgentSuiteRunFieldsFragment;
export type AgentQualityPoint = AgentQualityPointFieldsFragment;
export type AgentWorstRatedAnswer = AgentWorstRatedAnswerFieldsFragment;
export type AgentQualityControl = AgentQualityControlFieldsFragment;
export type AgentQualityOverview = AgentQualityOverviewQuery["agentQualityOverview"];
export type AgentSuiteRunCase = AgentSuiteRunCaseFieldsFragment;
export type { AgentSuiteRunStatus, UpdateAgentQualityControlInput };

type RequestOptions = { signal?: AbortSignal };

/** How far back the quality figures read, in days. */
export const QUALITY_WINDOW_DAYS = 30;

export const AGENT_QUALITY_LIST_KEY = "agent-quality-list";
export const AGENT_WORST_RATED_LIST_KEY = "agent-worst-rated-list";
export const AGENT_SUITE_RUN_LIST_KEY = "agent-suite-run-list";
export const AGENT_SUITE_RUN_CASE_LIST_KEY = "agent-suite-run-case-list";

/** Every agent with how people rate it and how it scores over the window. */
export const agentQualityTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentQualityAgentTableDocument,
  operationName: "AgentQualityAgentTable",
  connectionKey: "agentQualityAgentConnection",
  extraVariables: { window: QUALITY_WINDOW_DAYS },
});

export type AgentQualityRow = DataTableConfigRow<typeof agentQualityTableGraphQLConfig>;

/** The answers people liked least, for one agent or all of them. */
export function createAgentWorstRatedTableGraphQLConfig(agentDefinitionId: string | null) {
  return defineDataTableGraphQLConfig({
    document: AgentWorstRatedAnswerTableDocument,
    operationName: "AgentWorstRatedAnswerTable",
    connectionKey: "agentWorstRatedAnswerConnection",
    extraVariables: agentDefinitionId
      ? { agentDefinitionId, window: QUALITY_WINDOW_DAYS }
      : { window: QUALITY_WINDOW_DAYS },
  });
}

export type AgentWorstRatedRow = DataTableConfigRow<
  ReturnType<typeof createAgentWorstRatedTableGraphQLConfig>
>;

/** Suite runs, newest first, for one agent or all of them. */
export function createAgentSuiteRunTableGraphQLConfig(agentDefinitionId: string | null) {
  return defineDataTableGraphQLConfig({
    document: AgentSuiteRunTableDocument,
    operationName: "AgentSuiteRunTable",
    connectionKey: "agentSuiteRunConnection",
    extraVariables: agentDefinitionId ? { agentDefinitionId } : undefined,
  });
}

export type AgentSuiteRunRow = DataTableConfigRow<
  ReturnType<typeof createAgentSuiteRunTableGraphQLConfig>
>;

/** One suite run's cases, in the order they were asked unless sorted. */
export function createAgentSuiteRunCaseTableGraphQLConfig(suiteRunId: string) {
  return defineDataTableGraphQLConfig({
    document: AgentSuiteRunCaseTableDocument,
    operationName: "AgentSuiteRunCaseTable",
    connectionKey: "agentSuiteRunCaseConnection",
    extraVariables: { suiteRunId },
  });
}

export type AgentSuiteRunCaseRow = DataTableConfigRow<
  ReturnType<typeof createAgentSuiteRunCaseTableGraphQLConfig>
>;

export type AgentQualityDetail = {
  agentDefinitionId: string;
  agentName: string;
  enabled: boolean;
  windowDays: number;
  ratingsVisible: boolean;
  satisfaction: number | null;
  ratings: number;
  activeCases: number;
  satisfactionPoints: {
    day: string;
    positive: number;
    negative: number;
    satisfaction?: number | null;
  }[];
  qualityPoints: AgentQualityPoint[];
  worstRated: AgentWorstRatedAnswer[];
  lastSuiteRun: AgentSuiteRun | null;
};

export async function fetchAgentQualityOverview(
  options?: RequestOptions,
): Promise<AgentQualityOverview> {
  const data = await requestGraphQL({
    document: AgentQualityOverviewDocument,
    operationName: "AgentQualityOverview",
    variables: { window: QUALITY_WINDOW_DAYS },
    signal: options?.signal,
  });

  return data.agentQualityOverview;
}

export async function fetchAgentQuality(
  agentDefinitionId: string,
  options?: RequestOptions,
): Promise<AgentQualityDetail> {
  const data = await requestGraphQL({
    document: AgentQualityDocument,
    operationName: "AgentQuality",
    variables: { agentDefinitionId, window: QUALITY_WINDOW_DAYS },
    signal: options?.signal,
  });
  const detail = data.agentQuality;

  return {
    agentDefinitionId: detail.agentDefinitionId,
    agentName: detail.agentName,
    enabled: detail.enabled,
    windowDays: detail.windowDays,
    ratingsVisible: detail.ratingsVisible,
    satisfaction: detail.satisfaction ?? null,
    ratings: detail.ratings,
    activeCases: detail.activeCases,
    satisfactionPoints: detail.satisfactionPoints,
    qualityPoints: detail.qualityPoints.map((point) =>
      getFragmentData(AgentQualityPointFieldsFragmentDoc, point),
    ),
    worstRated: detail.worstRated.map((answer) =>
      getFragmentData(AgentWorstRatedAnswerFieldsFragmentDoc, answer),
    ),
    lastSuiteRun: detail.lastSuiteRun
      ? getFragmentData(AgentSuiteRunFieldsFragmentDoc, detail.lastSuiteRun)
      : null,
  };
}

/** One suite run, or null when it is not the organization's or no longer exists. */
export async function fetchAgentSuiteRun(
  id: string,
  options?: RequestOptions,
): Promise<AgentSuiteRun | null> {
  const data = await requestGraphQL({
    document: AgentSuiteRunDocument,
    operationName: "AgentSuiteRun",
    variables: { id },
    signal: options?.signal,
  });

  return data.agentSuiteRun
    ? getFragmentData(AgentSuiteRunFieldsFragmentDoc, data.agentSuiteRun)
    : null;
}

export async function fetchAgentQualityControl(
  options?: RequestOptions,
): Promise<AgentQualityControl> {
  const data = await requestGraphQL({
    document: AgentQualityControlDocument,
    operationName: "AgentQualityControl",
    variables: {},
    signal: options?.signal,
  });

  return getFragmentData(AgentQualityControlFieldsFragmentDoc, data.agentQualityControl);
}

export async function updateAgentQualityControl(
  input: UpdateAgentQualityControlInput,
): Promise<AgentQualityControl> {
  const data = await requestGraphQL({
    document: UpdateAgentQualityControlDocument,
    operationName: "UpdateAgentQualityControl",
    variables: { input },
  });

  return getFragmentData(AgentQualityControlFieldsFragmentDoc, data.updateAgentQualityControl);
}

export async function runAgentSuite(agentDefinitionId: string): Promise<AgentSuiteRun> {
  const data = await requestGraphQL({
    document: RunAgentSuiteDocument,
    operationName: "RunAgentSuite",
    variables: { agentDefinitionId },
  });

  return getFragmentData(AgentSuiteRunFieldsFragmentDoc, data.runAgentSuite);
}
