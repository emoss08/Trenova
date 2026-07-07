import { EdiSummaryDocument } from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { listEdiTemplatesGraphQL, type ListEdiTemplatesParams } from "@/lib/graphql/edi-templates";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export type EDITemplateListFilters = Omit<ListEdiTemplatesParams, "first" | "after"> & {
  limit?: number;
};

export const edi = createQueryKeys("edi", {
  connections: (query = "") => ({
    queryKey: ["connections", query],
    queryFn: async () => apiService.ediService.listConnections(query),
  }),
  mappingProfile: (partnerId: string) => ({
    queryKey: ["mapping-profile", partnerId],
    queryFn: async () => apiService.ediService.getMappingProfile(partnerId),
  }),
  mappingPreview: (transferId: string) => ({
    queryKey: ["mapping-preview", transferId],
    queryFn: async () => apiService.ediService.getMappingPreview(transferId),
  }),
  templates: (filters: EDITemplateListFilters = {}) => ({
    queryKey: ["templates", filters],
    queryFn: async () =>
      listEdiTemplatesGraphQL({
        first: filters.limit ?? 100,
        query: filters.query,
        status: filters.status,
        transactionSet: filters.transactionSet,
        direction: filters.direction,
      }),
  }),
  template: (templateId: string) => ({
    queryKey: ["template", templateId],
    queryFn: async () => apiService.ediService.getTemplate(templateId),
  }),
  templateVersions: (templateId: string) => ({
    queryKey: ["template-versions", templateId],
    queryFn: async () => apiService.ediService.listTemplateVersions(templateId),
  }),
  templateVersion: (templateId: string, versionId: string) => ({
    queryKey: ["template-version", templateId, versionId],
    queryFn: async () => apiService.ediService.getTemplateVersion(templateId, versionId),
  }),
  documentProfiles: (query = "") => ({
    queryKey: ["document-profiles", query],
    queryFn: async () => apiService.ediService.listPartnerDocumentProfiles(query),
  }),
  inboundFile: (fileId: string) => ({
    queryKey: ["inbound-file", fileId],
    queryFn: () => apiService.ediService.getInboundFile(fileId),
  }),
  messages: (query = "") => ({
    queryKey: ["messages", query],
    queryFn: async () => apiService.ediService.listMessages(query),
  }),
  message: (messageId: string) => ({
    queryKey: ["message", messageId],
    queryFn: async () => apiService.ediService.getMessage(messageId),
  }),
  messageInspection: (messageId: string) => ({
    queryKey: ["message-inspection", messageId],
    queryFn: async () => apiService.ediService.inspectMessage(messageId),
  }),
  testCases: (query = "") => ({
    queryKey: ["test-cases", query],
    queryFn: async () => apiService.ediService.listTestCases(query),
  }),
  testCase: (testCaseId: string) => ({
    queryKey: ["test-case", testCaseId],
    queryFn: async () => apiService.ediService.getTestCase(testCaseId),
  }),
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () =>
      requestGraphQL({
        document: EdiSummaryDocument,
        operationName: "EdiSummary",
        variables: { sinceHours: null },
      }),
  }),
});
