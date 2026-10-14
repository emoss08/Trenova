import { DESK_MEMORIES_KEY, fetchDeskMemorySettings } from "@/lib/graphql/desk-memories";

type Signal = { signal?: AbortSignal };

/** The Desk memory page's settings, under the root every Desk memory write invalidates. */
export function deskMemorySettingsQueryOptions() {
  return {
    queryKey: [DESK_MEMORIES_KEY, "settings"] as const,
    queryFn: ({ signal }: Signal) => fetchDeskMemorySettings(signal),
  };
}
