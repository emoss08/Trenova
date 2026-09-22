import { fetchTodaysBriefing } from "@/lib/graphql/briefing";
import type { BriefingRoleKey } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const briefing = createQueryKeys("briefing", {
  today: (roleKey?: BriefingRoleKey) => ({
    queryKey: [roleKey ?? "general"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchTodaysBriefing(roleKey, { signal }),
  }),
});
