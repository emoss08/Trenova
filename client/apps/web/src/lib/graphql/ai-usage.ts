import { AiUsageSummaryDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { AiUsageSummaryQuery } from "@trenova/graphql/generated/graphql";

type RequestOptions = { signal?: AbortSignal };

export type AIUsageSummary = AiUsageSummaryQuery["aiUsageSummary"];

const DAY_SECONDS = 24 * 60 * 60;

/** What the organization's models did over the last `days` days. */
export async function fetchAIUsageSummary(
  days: number,
  options?: RequestOptions,
): Promise<AIUsageSummary> {
  const since = Math.floor(Date.now() / 1000) - days * DAY_SECONDS;
  const data = await requestGraphQL({
    document: AiUsageSummaryDocument,
    operationName: "AIUsageSummary",
    variables: { since },
    signal: options?.signal,
  });

  return data.aiUsageSummary;
}
