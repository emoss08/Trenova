import {
  fetchAIControlSummary,
  fetchAITuneUps,
  fetchAIUsageDaily,
  fetchAgentPromotionPreview,
  fetchAgentRoster,
  fetchWorkingRuns,
} from "@/lib/graphql/ai-control";
import { fetchAgentActivitySummary } from "@/lib/graphql/agent-activity";
import type { AiControlTab } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const aiControl = createQueryKeys("aiControl", {
  summary: (tab: AiControlTab) => ({
    queryKey: [tab],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIControlSummary(tab, { signal }),
  }),
  activity: (since: number) => ({
    queryKey: [since],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentActivitySummary(since, { signal }),
  }),
  promotionPreview: (threshold: number) => ({
    queryKey: [threshold],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentPromotionPreview(threshold, { signal }),
  }),
  workingRuns: () => ({
    queryKey: ["working"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchWorkingRuns({ signal }),
  }),
  roster: () => ({
    queryKey: ["roster"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentRoster({ signal }),
  }),
  tuneUps: () => ({
    queryKey: ["tune-ups"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAITuneUps({ signal }),
  }),
  daily: (days: number, timezone: string) => ({
    queryKey: [days, timezone],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAIUsageDaily(days, timezone, { signal }),
  }),
});
