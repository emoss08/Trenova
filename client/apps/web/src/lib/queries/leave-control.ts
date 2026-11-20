import { fetchLeaveControl } from "@/lib/graphql/worker-leave";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const leaveControl = createQueryKeys("leaveControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchLeaveControl({ signal }),
  }),
});
