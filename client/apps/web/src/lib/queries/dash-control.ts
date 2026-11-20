import { fetchDashControl } from "@/lib/graphql/dash-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

// The root is the one the realtime map invalidates for `dash_control` (realtime-patching.ts).
export const dashControl = createQueryKeys("dash-control", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchDashControl({ signal }),
  }),
});
