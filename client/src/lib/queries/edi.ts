import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const edi = createQueryKeys("edi", {
  partners: (query = "") => ({
    queryKey: ["partners", query],
    queryFn: async () => apiService.ediService.listPartners(query),
  }),
  inboundTransfers: (query = "") => ({
    queryKey: ["transfers", "inbound", query],
    queryFn: async () => apiService.ediService.listInboundTransfers(query),
  }),
  outboundTransfers: (query = "") => ({
    queryKey: ["transfers", "outbound", query],
    queryFn: async () => apiService.ediService.listOutboundTransfers(query),
  }),
  partnerOptions: () => ({
    queryKey: ["partner-options"],
    queryFn: async () => apiService.ediService.selectPartners(),
  }),
  mappingProfile: (partnerId: string) => ({
    queryKey: ["mapping-profile", partnerId],
    queryFn: async () => apiService.ediService.getMappingProfile(partnerId),
  }),
  mappingPreview: (transferId: string) => ({
    queryKey: ["mapping-preview", transferId],
    queryFn: async () => apiService.ediService.getMappingPreview(transferId),
  }),
});
