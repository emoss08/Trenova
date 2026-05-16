import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const edi = createQueryKeys("edi", {
  partners: (query = "") => ({
    queryKey: ["partners", query],
    queryFn: async () => apiService.ediService.listPartners(query),
  }),
  connections: (query = "") => ({
    queryKey: ["connections", query],
    queryFn: async () => apiService.ediService.listConnections(query),
  }),
  communicationProfiles: (query = "") => ({
    queryKey: ["communication-profiles", query],
    queryFn: async () => apiService.ediService.listCommunicationProfiles(query),
  }),
  mappingProfiles: (query = "") => ({
    queryKey: ["mapping-profiles", query],
    queryFn: async () => apiService.ediService.listMappingProfiles(query),
  }),
  inboundTransfers: (query = "") => ({
    queryKey: ["transfers", "inbound", query],
    queryFn: async () => apiService.ediService.listInboundTransfers(query),
  }),
  outboundTransfers: (query = "") => ({
    queryKey: ["transfers", "outbound", query],
    queryFn: async () => apiService.ediService.listOutboundTransfers(query),
  }),
  shipmentLinks: (query = "") => ({
    queryKey: ["shipment-links", query],
    queryFn: async () => apiService.ediService.listShipmentLinks(query),
  }),
  transferChanges: (query = "") => ({
    queryKey: ["transfer-changes", query],
    queryFn: async () => apiService.ediService.listTransferChanges(query),
  }),
  partnerOptions: () => ({
    queryKey: ["partner-options"],
    queryFn: async () => apiService.ediService.selectPartners(),
  }),
  mappingProfile: (partnerId: string) => ({
    queryKey: ["mapping-profile", partnerId],
    queryFn: async () => apiService.ediService.getMappingProfile(partnerId),
  }),
  mappingProfileById: (profileId: string) => ({
    queryKey: ["mapping-profile-by-id", profileId],
    queryFn: async () => apiService.ediService.getMappingProfileById(profileId),
  }),
  mappingPreview: (transferId: string) => ({
    queryKey: ["mapping-preview", transferId],
    queryFn: async () => apiService.ediService.getMappingPreview(transferId),
  }),
  documentTypes: () => ({
    queryKey: ["document-types"],
    queryFn: async () => apiService.ediService.listDocumentTypes(),
  }),
  templates: (query = "") => ({
    queryKey: ["templates", query],
    queryFn: async () => apiService.ediService.listTemplates(query),
  }),
  documentProfiles: (query = "") => ({
    queryKey: ["document-profiles", query],
    queryFn: async () => apiService.ediService.listPartnerDocumentProfiles(query),
  }),
  messages: (query = "") => ({
    queryKey: ["messages", query],
    queryFn: async () => apiService.ediService.listMessages(query),
  }),
  message: (messageId: string) => ({
    queryKey: ["message", messageId],
    queryFn: async () => apiService.ediService.getMessage(messageId),
  }),
  testCases: (query = "") => ({
    queryKey: ["test-cases", query],
    queryFn: async () => apiService.ediService.listTestCases(query),
  }),
});
