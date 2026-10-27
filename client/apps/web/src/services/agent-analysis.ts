import { translate } from "@trenova/shared/i18n/runtime";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { analysisRunSchema, analysisRunsResponseSchema } from "@/types/analysis";

/** Where a page of the history starts: just after the last analysis of the page before. */
export type AnalysisRunsCursor = { before: number; beforeId: string };

export class AgentAnalysisService {
  /** The organization's analyses, newest first, for those who oversee agents. */
  public async listRuns(
    params: { cursor?: AnalysisRunsCursor | null; limit?: number },
    options?: { signal?: AbortSignal },
  ) {
    const query = new URLSearchParams();
    if (params.cursor) {
      query.set("before", String(params.cursor.before));
      query.set("beforeId", params.cursor.beforeId);
    }
    if (params.limit) query.set("limit", String(params.limit));
    const suffix = query.size > 0 ? `?${query.toString()}` : "";
    const response = await api.get(`/agent-analysis/runs/${suffix}`, { signal: options?.signal });
    return safeParse(analysisRunsResponseSchema, response, translate("Analyses"));
  }

  /** One analysis, whole: the Desk reads it for a table longer than the agent was shown. */
  public async run(runId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/agent-analysis/runs/${encodeURIComponent(runId)}/`, {
      signal: options?.signal,
    });
    return safeParse(analysisRunSchema, response, translate("Analysis"));
  }
}
