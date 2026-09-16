import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { insightListSchema, insightSchema, type Insight } from "@/types/insight";

export class InsightService {
  /**
   * Insights are read, never generated on demand. Refreshing them runs several
   * month-wide aggregates and may call a model, so it happens on a schedule and
   * this only fetches what that produced.
   */
  public async list(limit = 5) {
    const response = await api.get(`/insights/?limit=${limit}`);
    return safeParse(insightListSchema, response, "Insight");
  }

  public async dismiss(id: Insight["id"], reason = "") {
    const response = await api.post(`/insights/${id}/dismiss/`, { reason });
    return safeParse(insightSchema, response, "Insight");
  }
}
