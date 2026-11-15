import {
  fetchCurrentSupportSession,
  fetchSupportAccessSettings,
} from "@/lib/graphql/support-access";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const supportAccess = createQueryKeys("supportAccess", {
  settings: () => ({
    queryKey: ["settings"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchSupportAccessSettings({ signal }),
  }),
  grants: () => ({
    queryKey: ["grants"],
  }),
  actions: (grantId: string) => ({
    queryKey: ["actions", grantId],
  }),
  /** Read once a session: a session either is Trenova support's or is not, for its whole life. */
  currentSession: () => ({
    queryKey: ["currentSession"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchCurrentSupportSession({ signal }),
  }),
});
