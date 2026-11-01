import {
  AgentAnalysisRunDocument,
  AgentAnalysisRunsDocument,
} from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
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
    const data = await requestGraphQL({
      document: AgentAnalysisRunsDocument,
      operationName: "AgentAnalysisRuns",
      variables: {
        before: params.cursor?.before ?? null,
        beforeId: params.cursor?.beforeId ?? null,
        limit: params.limit ?? null,
      },
      signal: options?.signal,
    });
    return safeParse(analysisRunsResponseSchema, data.agentAnalysisRuns, translate("Analyses"));
  }

  /** One analysis, whole: the Desk reads it for a table longer than the agent was shown. */
  public async run(runId: string, options?: { signal?: AbortSignal }) {
    const data = await requestGraphQL({
      document: AgentAnalysisRunDocument,
      operationName: "AgentAnalysisRun",
      variables: { id: runId },
      signal: options?.signal,
    });
    return safeParse(analysisRunSchema, data.agentAnalysisRun, translate("Analysis"));
  }
}
