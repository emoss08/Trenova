import { createQueryKeys } from "@lukemorales/query-key-factory";
import { supportAccessService } from "../../services/support-access";

export const supportAccess = createQueryKeys("supportAccess", {
  grantState: () => ({
    queryKey: ["grantState"],
    queryFn: async () => supportAccessService.grantState(),
  }),
  staffProfile: () => ({
    queryKey: ["staffProfile"],
    queryFn: async () => supportAccessService.staffProfile(),
  }),
  grantedOrganizations: () => ({
    queryKey: ["grantedOrganizations"],
    queryFn: async () => supportAccessService.grantedOrganizations(),
  }),
  currentSession: () => ({
    queryKey: ["currentSession"],
    queryFn: async () => supportAccessService.currentSession(),
  }),
});
