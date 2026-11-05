import { agentAnalysisService } from "@/services/agent-analysis";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** How many analyses the history shows at once. */
export const ANALYSIS_RUNS_PAGE = 25;

export const agentAnalysis = createQueryKeys("agentAnalysis", {
  // The organization's analyses, newest first, paged by useInfiniteQuery.
  runs: () => ({
    queryKey: ["runs"],
  }),
  run: (runId: string) => ({
    queryKey: ["run", runId],
    queryFn: ({ signal }: { signal?: AbortSignal }) => agentAnalysisService.run(runId, { signal }),
  }),
});
