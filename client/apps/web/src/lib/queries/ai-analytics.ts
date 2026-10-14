import {
  fetchAgentAnalytics,
  fetchAgentAnalyticsToolFailures,
  type AIAnalyticsInput,
} from "@/lib/graphql/ai-analytics";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const aiAnalytics = createQueryKeys("aiAnalytics", {
  models: (input: AIAnalyticsInput) => ({
    queryKey: [input],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentAnalytics(input, { signal }),
  }),
  toolFailures: (input: AIAnalyticsInput, seriesKey: string, toolName: string) => ({
    queryKey: [input, seriesKey, toolName],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentAnalyticsToolFailures(input, seriesKey, toolName, { signal }),
  }),
});
