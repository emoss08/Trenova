import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  insightDetailSchema,
  insightListSchema,
  insightPageSchema,
  insightSchema,
  type Insight,
  type InsightCategory,
  type InsightSeverity,
  type InsightSurface,
} from "@/types/insight";

export type BrowseInsightsParams = {
  statuses: string[];
  categories: InsightCategory[];
  severities: InsightSeverity[];
  limit: number;
  offset: number;
};

/**
 * The active-insights read, as a URL.
 *
 * A page names the surface it is asking for; the home widget names none and
 * gets the whole view. The parameter is only written when given so the
 * unfiltered request says nothing it does not mean.
 */
export function activeInsightsPath(limit: number, surface?: InsightSurface): string {
  const query = new URLSearchParams({ limit: String(limit) });
  if (surface) {
    query.set("surface", surface);
  }

  return `/insights/?${query.toString()}`;
}

export class InsightService {
  /**
   * Insights are read, never generated on demand. Refreshing them runs several
   * month-wide aggregates and may call a model, so it happens on a schedule and
   * this only fetches what that produced.
   */
  public async list(limit = 5, surface?: InsightSurface) {
    const response = await api.get(activeInsightsPath(limit, surface));
    return safeParse(insightListSchema, response, "Insight");
  }

  /**
   * Browsing is a different endpoint from the widget's read, not the same one
   * with a flag: this one defaults to nothing and is told exactly which
   * lifecycle to show, so it can never quietly hand back dismissed findings to
   * a caller that did not ask for them.
   */
  public async browse(params: BrowseInsightsParams) {
    const query = new URLSearchParams();

    for (const status of params.statuses) {
      query.append("status", status);
    }
    for (const category of params.categories) {
      query.append("category", category);
    }
    for (const severity of params.severities) {
      query.append("severity", severity);
    }
    query.set("limit", String(params.limit));
    query.set("offset", String(params.offset));

    const response = await api.get(`/insights/browse/?${query.toString()}`);
    return safeParse(insightPageSchema, response, "Insight");
  }

  public async dismiss(id: Insight["id"], reason = "") {
    const response = await api.post(`/insights/${id}/dismiss/`, { reason });
    return safeParse(insightSchema, response, "Insight");
  }

  /**
   * One finding with its trend and the rule behind it.
   *
   * A finding the reader may not see comes back as a 404 rather than a 403:
   * saying "this exists but is not for you" about a card naming a customer and
   * a revenue figure is itself a disclosure.
   */
  public async detail(id: Insight["id"]) {
    const response = await api.get(`/insights/${id}/`);
    return safeParse(insightDetailSchema, response, "Insight");
  }

  /** Undoes a dismissal, so a misclick is not a month-long mistake. */
  public async restore(id: Insight["id"]) {
    const response = await api.post(`/insights/${id}/restore/`, {});
    return safeParse(insightSchema, response, "Insight");
  }
}
