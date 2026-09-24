import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentQualityAgentsDocument,
  AgentQualityControlDocument,
  AgentQualityControlFieldsFragmentDoc,
  AgentQualityDocument,
  AgentQualityOverviewDocument,
  AgentQualityPointFieldsFragmentDoc,
  AgentSuiteRunCasesDocument,
  AgentSuiteRunFieldsFragmentDoc,
  AgentSuiteRunsDocument,
  AgentWorstRatedAnswerFieldsFragmentDoc,
  AgentWorstRatedAnswersDocument,
  DataTablePageInfoFieldsFragmentDoc,
  RunAgentSuiteDocument,
  UpdateAgentQualityControlDocument,
  type AgentQualityControlFieldsFragment,
  type AgentQualityOverviewQuery,
  type AgentQualityPointFieldsFragment,
  type AgentSuiteRunCasesQuery,
  type AgentSuiteRunFieldsFragment,
  type AgentSuiteRunStatus,
  type AgentWorstRatedAnswerFieldsFragment,
  type UpdateAgentQualityControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentSuiteRun = AgentSuiteRunFieldsFragment;
export type AgentQualityPoint = AgentQualityPointFieldsFragment;
export type AgentWorstRatedAnswer = AgentWorstRatedAnswerFieldsFragment;
export type AgentQualityControl = AgentQualityControlFieldsFragment;
export type AgentQualityOverview = AgentQualityOverviewQuery["agentQualityOverview"];
export type AgentSuiteRunCase =
  AgentSuiteRunCasesQuery["agentSuiteRunCases"]["edges"][number]["node"];
export type { AgentSuiteRunStatus, UpdateAgentQualityControlInput };

type RequestOptions = { signal?: AbortSignal };

/** How far back the quality figures read, in days. */
export const QUALITY_WINDOW_DAYS = 30;

/** One page a section asks the server for; the count is read only on the first. */
export type QualityPageRequest = {
  first: number;
  after: string | null;
  includeTotalCount: boolean;
};

export type QualityPage<T> = {
  items: T[];
  endCursor: string | null;
  hasNextPage: boolean;
  totalCount: number | null;
};

export type AgentQualityRow = {
  agentDefinitionId: string;
  name: string;
  enabled: boolean;
  ratingsVisible: boolean;
  satisfaction: number | null;
  satisfactionDelta: number | null;
  ratings: number;
  qualityScore: number | null;
  openRegression: boolean;
  qualityPoints: AgentQualityPoint[];
  lastSuiteRun: AgentSuiteRun | null;
};

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

function afterOf(after: string | null): { after?: string } {
  return after ? { after } : {};
}

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

export async function fetchAgentQualityAgents(
  page: QualityPageRequest,
  options?: RequestOptions,
): Promise<QualityPage<AgentQualityRow>> {
  const data = await requestGraphQL({
    document: AgentQualityAgentsDocument,
    operationName: "AgentQualityAgents",
    variables: {
      input: { window: QUALITY_WINDOW_DAYS, first: page.first, ...afterOf(page.after) },
      includeTotalCount: page.includeTotalCount,
    },
    signal: options?.signal,
  });
  const connection = data.agentQualityAgents;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map(({ node }) => ({
      agentDefinitionId: node.agentDefinitionId,
      name: node.name,
      enabled: node.enabled,
      ratingsVisible: node.ratingsVisible,
      satisfaction: node.satisfaction ?? null,
      satisfactionDelta: node.satisfactionDelta ?? null,
      ratings: node.ratings,
      qualityScore: node.qualityScore ?? null,
      openRegression: node.openRegression,
      qualityPoints: node.qualityPoints.map((point) =>
        getFragmentData(AgentQualityPointFieldsFragmentDoc, point),
      ),
      lastSuiteRun: node.lastSuiteRun
        ? getFragmentData(AgentSuiteRunFieldsFragmentDoc, node.lastSuiteRun)
        : null,
    })),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
}

export async function fetchAgentWorstRatedAnswers(
  agentDefinitionId: string | null,
  page: QualityPageRequest,
  options?: RequestOptions,
): Promise<QualityPage<AgentWorstRatedAnswer>> {
  const data = await requestGraphQL({
    document: AgentWorstRatedAnswersDocument,
    operationName: "AgentWorstRatedAnswers",
    variables: {
      input: {
        window: QUALITY_WINDOW_DAYS,
        first: page.first,
        ...afterOf(page.after),
        ...(agentDefinitionId ? { agentDefinitionId } : {}),
      },
    },
    signal: options?.signal,
  });
  const connection = data.agentWorstRatedAnswers;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map(({ node }) =>
      getFragmentData(AgentWorstRatedAnswerFieldsFragmentDoc, node),
    ),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: null,
  };
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

export async function fetchAgentSuiteRuns(
  agentDefinitionId: string,
  page: QualityPageRequest,
  options?: RequestOptions,
): Promise<QualityPage<AgentSuiteRun>> {
  const data = await requestGraphQL({
    document: AgentSuiteRunsDocument,
    operationName: "AgentSuiteRuns",
    variables: {
      input: { agentDefinitionId, first: page.first, ...afterOf(page.after) },
      includeTotalCount: page.includeTotalCount,
    },
    signal: options?.signal,
  });
  const connection = data.agentSuiteRuns;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map(({ node }) =>
      getFragmentData(AgentSuiteRunFieldsFragmentDoc, node),
    ),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
}

export async function fetchAgentSuiteRunCases(
  suiteRunId: string,
  page: QualityPageRequest,
  options?: RequestOptions,
): Promise<QualityPage<AgentSuiteRunCase>> {
  const data = await requestGraphQL({
    document: AgentSuiteRunCasesDocument,
    operationName: "AgentSuiteRunCases",
    variables: {
      input: { suiteRunId, first: page.first, ...afterOf(page.after) },
      includeTotalCount: page.includeTotalCount,
    },
    signal: options?.signal,
  });
  const connection = data.agentSuiteRunCases;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map(({ node }) => node),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
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
