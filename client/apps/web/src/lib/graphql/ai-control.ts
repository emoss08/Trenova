import type { AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import {
  AgentPromotionPreviewDocument,
  AiAgentRosterDocument,
  AgentRunTableDocument,
  AgentRunTableRowFieldsFragmentDoc,
  AiControlSummaryDocument,
  AiTuneUpFieldsFragmentDoc,
  AiTuneUpsDocument,
  ApplyAiTuneUpDocument,
  DismissAiTuneUpDocument,
  RestoreAiTuneUpDocument,
  AiProviderFailureFieldsFragmentDoc,
  AiUsageDailyDocument,
  DismissAiProviderFailureDocument,
  RestoreAiProviderFailureDocument,
  type AgentPromotionPreviewQuery,
  type AiAgentRosterQuery,
  type AiControlSummaryQuery,
  type AiControlTab,
  type AiProviderFailureFieldsFragment,
  type AiTuneUpFieldsFragment,
  type AiUsageDailyQuery,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

type RawSummary = AiControlSummaryQuery["aiControlSummary"];

export type AIProviderFailure = AiProviderFailureFieldsFragment;
export type AIControlSegment = RawSummary["segments"][number];
export type AIControlSummary = Omit<RawSummary, "facts" | "visibleFailures"> & {
  facts: Omit<RawSummary["facts"], "failing"> & { failing: AIProviderFailure[] };
  visibleFailures: AIProviderFailure[];
};
export type ToolPromotion = AgentPromotionPreviewQuery["agentPromotionPreview"][number];
export type AIUsageDay = AiUsageDailyQuery["aiUsageDaily"][number];
export type AITuneUp = Omit<AiTuneUpFieldsFragment, " $fragmentName">;
export type AITuneUps = { items: AITuneUp[]; computedAt: number | null; windowDays: number };

export async function fetchAIControlSummary(
  tab: AiControlTab,
  options?: RequestOptions,
): Promise<AIControlSummary> {
  const data = await requestGraphQL({
    document: AiControlSummaryDocument,
    operationName: "AIControlSummary",
    variables: { tab },
    signal: options?.signal,
  });
  const summary = data.aiControlSummary;

  return {
    ...summary,
    facts: {
      ...summary.facts,
      failing: summary.facts.failing.map((failure) =>
        getFragmentData(AiProviderFailureFieldsFragmentDoc, failure),
      ),
    },
    visibleFailures: summary.visibleFailures.map((failure) =>
      getFragmentData(AiProviderFailureFieldsFragmentDoc, failure),
    ),
  };
}

export async function dismissAIProviderFailure(
  providerId: string,
  lastFailureAt: number,
): Promise<boolean> {
  const data = await requestGraphQL({
    document: DismissAiProviderFailureDocument,
    operationName: "DismissAIProviderFailure",
    variables: { providerId, lastFailureAt },
  });
  return data.dismissAIProviderFailure;
}

export async function restoreAIProviderFailure(providerId: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: RestoreAiProviderFailureDocument,
    operationName: "RestoreAIProviderFailure",
    variables: { providerId },
  });
  return data.restoreAIProviderFailure;
}

export async function fetchAgentPromotionPreview(
  threshold: number,
  options?: RequestOptions,
): Promise<ToolPromotion[]> {
  const data = await requestGraphQL({
    document: AgentPromotionPreviewDocument,
    operationName: "AgentPromotionPreview",
    variables: { threshold },
    signal: options?.signal,
  });
  return data.agentPromotionPreview;
}

export async function fetchAIUsageDaily(
  days: number,
  timezone: string,
  options?: RequestOptions,
): Promise<AIUsageDay[]> {
  const data = await requestGraphQL({
    document: AiUsageDailyDocument,
    operationName: "AIUsageDaily",
    variables: { days, timezone },
    signal: options?.signal,
  });
  return data.aiUsageDaily;
}

/** Run statuses of a run still going. */
const WORKING_STATUSES = ["Pending", "GatheringContext", "Diagnosing"] as const;
/** How many running agents the overview names; the rest are counted. */
const WORKING_RUNS_SHOWN = 12;

export type WorkingRun = Pick<
  AgentRunRow,
  "id" | "agentDefinitionId" | "status" | "summary" | "startedAt"
>;

/** The runs going right now, newest first. */
export async function fetchWorkingRuns(options?: RequestOptions): Promise<WorkingRun[]> {
  const data = await requestGraphQL({
    document: AgentRunTableDocument,
    operationName: "AgentRunTable",
    variables: {
      input: {
        first: WORKING_RUNS_SHOWN,
        fieldFilters: [{ field: "status", operator: "in", value: [...WORKING_STATUSES] }],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
      includeTotalCount: false,
    },
    signal: options?.signal,
  });

  return data.agentRuns.edges.map((edge) => {
    const row = getFragmentData(AgentRunTableRowFieldsFragmentDoc, edge.node);
    return {
      id: row.id,
      agentDefinitionId: row.agentDefinitionId,
      status: row.status,
      summary: row.summary,
      startedAt: row.startedAt,
    };
  });
}

export type AgentRosterStat = AiAgentRosterQuery["aiAgentRoster"][number];

/** What each agent has done lately, by agent id. */
export async function fetchAgentRoster(
  options?: RequestOptions,
): Promise<Map<string, AgentRosterStat>> {
  const data = await requestGraphQL({
    document: AiAgentRosterDocument,
    operationName: "AIAgentRoster",
    signal: options?.signal,
  });
  return new Map(data.aiAgentRoster.map((stat) => [stat.agentId, stat]));
}

export async function fetchAITuneUps(options?: RequestOptions): Promise<AITuneUps> {
  const data = await requestGraphQL({
    document: AiTuneUpsDocument,
    operationName: "AITuneUps",
    signal: options?.signal,
  });
  const { items, computedAt, windowDays } = data.aiTuneUps;
  return {
    computedAt,
    windowDays,
    items: items.map((item) => getFragmentData(AiTuneUpFieldsFragmentDoc, item)),
  };
}

export type TuneUpDecision = { id: string; version: number };

export async function applyAITuneUp({ id, version }: TuneUpDecision): Promise<AITuneUp> {
  const data = await requestGraphQL({
    document: ApplyAiTuneUpDocument,
    operationName: "ApplyAITuneUp",
    variables: { id, version },
  });
  return getFragmentData(AiTuneUpFieldsFragmentDoc, data.applyAITuneUp);
}

export async function dismissAITuneUp({ id, version }: TuneUpDecision): Promise<AITuneUp> {
  const data = await requestGraphQL({
    document: DismissAiTuneUpDocument,
    operationName: "DismissAITuneUp",
    variables: { id, version },
  });
  return getFragmentData(AiTuneUpFieldsFragmentDoc, data.dismissAITuneUp);
}

export async function restoreAITuneUp({ id, version }: TuneUpDecision): Promise<AITuneUp> {
  const data = await requestGraphQL({
    document: RestoreAiTuneUpDocument,
    operationName: "RestoreAITuneUp",
    variables: { id, version },
  });
  return getFragmentData(AiTuneUpFieldsFragmentDoc, data.restoreAITuneUp);
}
