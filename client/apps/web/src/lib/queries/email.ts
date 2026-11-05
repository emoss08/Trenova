import { emailService } from "@/services/email";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const email = createQueryKeys("email", {
  profiles: (params = "") => ({
    queryKey: ["profiles", params],
    queryFn: () => emailService.listProfiles(params),
  }),
  assignments: () => ({
    queryKey: ["assignments"],
    queryFn: () => emailService.listAssignments(),
  }),
  logs: (params = "") => ({
    queryKey: ["logs", params],
    queryFn: () => emailService.listLogs(params),
  }),
  suppressions: (params = "") => ({
    queryKey: ["suppressions", params],
    queryFn: () => emailService.listSuppressions(params),
  }),
});
