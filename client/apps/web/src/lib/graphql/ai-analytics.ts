import {
  AgentAnalyticsDocument,
  AgentAnalyticsToolFailuresDocument,
  AiAnalyticsProviderFieldsFragmentDoc,
  AiAnalyticsSegmentFieldsFragmentDoc,
  AiAnalyticsSeriesFieldsFragmentDoc,
  RouteAiTaskDocument,
  type AgentAnalyticsQuery,
  type AgentAnalyticsToolFailuresQuery,
  type AiAnalyticsInput,
  type AiAnalyticsProviderFieldsFragment,
  type AiAnalyticsSegmentFieldsFragment,
  type AiAnalyticsSeriesFieldsFragment,
  type AiTask,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

type RawAnalytics = AgentAnalyticsQuery["agentAnalytics"];
type RawTask = RawAnalytics["tasks"][number];
type RawProviderHealth = RawAnalytics["providers"][number];
type RawEvent = RawAnalytics["events"][number];
type RawSeries = Omit<AiAnalyticsSeriesFieldsFragment, " $fragmentName">;
type RawFailureDetail = AgentAnalyticsToolFailuresQuery["agentAnalyticsToolFailures"];

export type AIAnalyticsInput = AiAnalyticsInput;
export type AIAnalyticsProvider = Omit<AiAnalyticsProviderFieldsFragment, " $fragmentName">;
export type AIAnalyticsSegment = Omit<AiAnalyticsSegmentFieldsFragment, " $fragmentName">;
/** A model, a provider, or a deployment (one model on one provider). */
export type AIAnalyticsSeries = Omit<RawSeries, "providers"> & { providers: AIAnalyticsProvider[] };
export type AIAnalyticsRunTime = RawSeries["runTimes"][number];
export type AIAnalyticsCauseCount = RawSeries["toolFailuresByCause"][number];
export type AIAnalyticsTrendPoint = RawAnalytics["trend"][number];
export type AIAnalyticsTool = RawAnalytics["tools"][number];
export type AIAnalyticsEvent = Omit<RawEvent, "provider"> & { provider: AIAnalyticsProvider };
export type AIAnalyticsTaskRow = Omit<RawTask, "current" | "suggestion"> & {
  current: AIAnalyticsProvider | null;
  suggestion:
    | (Omit<NonNullable<RawTask["suggestion"]>, "provider"> & {
        provider: AIAnalyticsProvider;
      })
    | null;
};
export type AIAnalyticsProviderHealth = Omit<RawProviderHealth, "provider"> & {
  provider: AIAnalyticsProvider;
};
export type AIAnalyticsTotals = RawAnalytics["totals"];
export type AIAnalyticsSuggestions = NonNullable<RawAnalytics["suggestions"]>;
export type AIAnalyticsAgentSuggestions = AIAnalyticsSuggestions["agents"][number];
export type AgentAnalytics = Omit<
  RawAnalytics,
  "summary" | "runTimeNote" | "series" | "deployments" | "events" | "tasks" | "providers"
> & {
  summary: AIAnalyticsSegment[];
  runTimeNote: AIAnalyticsSegment[];
  series: AIAnalyticsSeries[];
  deployments: AIAnalyticsSeries[];
  events: AIAnalyticsEvent[];
  tasks: AIAnalyticsTaskRow[];
  providers: AIAnalyticsProviderHealth[];
};
export type AgentAnalyticsToolFailures = Omit<RawFailureDetail, "providers"> & {
  providers: AIAnalyticsProvider[];
};

function provider(
  ref: FragmentType<typeof AiAnalyticsProviderFieldsFragmentDoc>,
): AIAnalyticsProvider {
  return getFragmentData(AiAnalyticsProviderFieldsFragmentDoc, ref);
}

function segment(
  ref: FragmentType<typeof AiAnalyticsSegmentFieldsFragmentDoc>,
): AIAnalyticsSegment {
  return getFragmentData(AiAnalyticsSegmentFieldsFragmentDoc, ref);
}

function series(ref: FragmentType<typeof AiAnalyticsSeriesFieldsFragmentDoc>): AIAnalyticsSeries {
  const raw = getFragmentData(AiAnalyticsSeriesFieldsFragmentDoc, ref);

  return { ...raw, providers: raw.providers.map(provider) };
}

/** How the organization's models and providers compare over a window. */
export async function fetchAgentAnalytics(
  input: AIAnalyticsInput,
  options?: RequestOptions,
): Promise<AgentAnalytics> {
  const data = await requestGraphQL({
    document: AgentAnalyticsDocument,
    operationName: "AgentAnalytics",
    variables: { input },
    signal: options?.signal,
  });
  const analytics = data.agentAnalytics;

  return {
    ...analytics,
    summary: analytics.summary.map(segment),
    runTimeNote: analytics.runTimeNote.map(segment),
    series: analytics.series.map(series),
    deployments: analytics.deployments.map(series),
    events: analytics.events.map((event) => ({ ...event, provider: provider(event.provider) })),
    tasks: analytics.tasks.map((task) => ({
      ...task,
      current: task.current ? provider(task.current) : null,
      suggestion: task.suggestion
        ? { ...task.suggestion, provider: provider(task.suggestion.provider) }
        : null,
    })),
    providers: analytics.providers.map((health) => ({
      ...health,
      provider: provider(health.provider),
    })),
  };
}

/** Why one tool's calls failed for one model or provider, with the newest failures. */
export async function fetchAgentAnalyticsToolFailures(
  input: AIAnalyticsInput,
  seriesKey: string,
  toolName: string,
  options?: RequestOptions,
): Promise<AgentAnalyticsToolFailures> {
  const data = await requestGraphQL({
    document: AgentAnalyticsToolFailuresDocument,
    operationName: "AgentAnalyticsToolFailures",
    variables: { input, seriesKey, toolName },
    signal: options?.signal,
  });
  const detail = data.agentAnalyticsToolFailures;

  return { ...detail, providers: detail.providers.map(provider) };
}

/** Makes a provider the first to take a task, leaving every other task where it goes. */
export async function routeAITask(task: AiTask, providerId: string): Promise<void> {
  await requestGraphQL({
    document: RouteAiTaskDocument,
    operationName: "RouteAITask",
    variables: { task, providerId },
  });
}
