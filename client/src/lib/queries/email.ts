import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const email = createQueryKeys("email", {
  profiles: (params = "") => ({
    queryKey: ["profiles", params],
    queryFn: () => apiService.emailService.listProfiles(params),
  }),
  assignments: () => ({
    queryKey: ["assignments"],
    queryFn: () => apiService.emailService.listAssignments(),
  }),
  logs: (params = "") => ({
    queryKey: ["logs", params],
    queryFn: () => apiService.emailService.listLogs(params),
  }),
  suppressions: (params = "") => ({
    queryKey: ["suppressions", params],
    queryFn: () => apiService.emailService.listSuppressions(params),
  }),
});

