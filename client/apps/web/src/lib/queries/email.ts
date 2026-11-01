import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const email = createQueryKeys("email", {
  assignments: () => ({
    queryKey: ["assignments"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.emailService.listAssignments({ signal }),
  }),
  logs: () => ({
    queryKey: ["logs"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => apiService.emailService.listLogs({ signal }),
  }),
});
