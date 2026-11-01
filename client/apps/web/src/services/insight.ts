import {
  ActiveInsightsDocument,
  BrowseInsightsDocument,
  DismissInsightDocument,
  InsightDetailDocument,
  RestoreInsightDocument,
  type ActiveInsightsInput,
  type InsightStatus,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  insightDetailSchema,
  insightPageSchema,
  insightSchema,
  insightStatusSchema,
  type Insight,
  type InsightCategory,
  type InsightSeverity,
  type InsightSurface,
} from "@/types/insight";
import { z } from "zod";

export type BrowseInsightsParams = {
  statuses: string[];
  categories: InsightCategory[];
  severities: InsightSeverity[];
  limit: number;
  offset: number;
};

/**
 * The active-insights read's input.
 *
 * A page names the surface it is asking for; the home widget names none and
 * gets the whole view. The surface is only sent when given so the unfiltered
 * request says nothing it does not mean.
 */
export function activeInsightsInput(limit: number, surface?: InsightSurface): ActiveInsightsInput {
  return surface ? { limit, surface } : { limit };
}

function knownStatuses(statuses: string[]): InsightStatus[] {
  return statuses.filter(
    (status): status is InsightStatus => insightStatusSchema.safeParse(status).success,
  );
}

export class InsightService {
  /**
   * Insights are read, never generated on demand. Refreshing them runs several
   * month-wide aggregates and may call a model, so it happens on a schedule and
   * this only fetches what that produced.
   */
  public async list(limit = 5, surface?: InsightSurface, options: { signal?: AbortSignal } = {}) {
    const data = await requestGraphQL({
      document: ActiveInsightsDocument,
      operationName: "ActiveInsights",
      variables: { input: activeInsightsInput(limit, surface) },
      signal: options.signal,
    });
    const results = await safeParse(z.array(insightSchema), data.activeInsights, "Insight");
    return { results };
  }

  /**
   * Browsing is a different read from the widget's, not the same one with a
   * flag: this one defaults to nothing and is told exactly which lifecycle to
   * show, so it can never quietly hand back dismissed findings to a caller that
   * did not ask for them.
   */
  public async browse(params: BrowseInsightsParams, options: { signal?: AbortSignal } = {}) {
    const data = await requestGraphQL({
      document: BrowseInsightsDocument,
      operationName: "BrowseInsights",
      variables: {
        input: {
          statuses: knownStatuses(params.statuses),
          categories: params.categories,
          severities: params.severities,
          limit: params.limit,
          offset: params.offset,
        },
      },
      signal: options.signal,
    });
    return safeParse(insightPageSchema, data.insights, "Insight");
  }

  public async dismiss(id: Insight["id"], reason = "") {
    const data = await requestGraphQL({
      document: DismissInsightDocument,
      operationName: "DismissInsight",
      variables: { input: { insightId: id, reason } },
    });
    return safeParse(insightSchema, data.dismissInsight, "Insight");
  }

  /**
   * One finding with its trend and the rule behind it.
   *
   * A finding the reader may not see comes back as not found rather than
   * forbidden: saying "this exists but is not for you" about a card naming a
   * customer and a revenue figure is itself a disclosure.
   */
  public async detail(id: Insight["id"], options: { signal?: AbortSignal } = {}) {
    const data = await requestGraphQL({
      document: InsightDetailDocument,
      operationName: "InsightDetail",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(insightDetailSchema, data.insight, "Insight");
  }

  /** Undoes a dismissal, so a misclick is not a month-long mistake. */
  public async restore(id: Insight["id"]) {
    const data = await requestGraphQL({
      document: RestoreInsightDocument,
      operationName: "RestoreInsight",
      variables: { id },
    });
    return safeParse(insightSchema, data.restoreInsight, "Insight");
  }
}
