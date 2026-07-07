import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  ediBulkActionResultSchema,
  ediPartnerReadinessSchema,
  ediCertificateSummarySchema,
  ediConnectionTestResultSchema,
  ediMappingPreviewSchema,
  ediMappingProfileItemSchema,
  ediMappingProfileSchema,
  ediCommunicationProfileSchema,
  ediConnectionListSchema,
  ediConnectionSchema,
  ediDocumentPreviewSchema,
  ediMessageInspectionSchema,
  ediInboundFileSchema,
  ediMessageListSchema,
  ediMessageSchema,
  ediTestCaseListSchema,
  ediTestCaseSchema,
  ediPartnerDocumentProfileListSchema,
  ediPartnerDocumentProfileSchema,
  ediPartnerSchema,
  ediTemplateSchema,
  ediTemplateValidationResponseSchema,
  ediTemplateVersionSchema,
  ediTransferSchema,
  internalPartnerPairSchema,
  type ApproveEDITransferRequest,
  type CreateEDIConnectionRequest,
  type CreateEDITemplateDraftRequest,
  type CreateEDITemplateRequest,
  type CreateInternalPartnerPairRequest,
  type EDIConnectionActionRequest,
  type EDIMappingProfileItem,
  type EDITemplateActionRequest,
  type GenerateEDIDocumentRequest,
  type InspectX12Request,
  type PreviewEDIDocumentRequest,
  type ReplaceEDITemplateScriptLibrariesRequest,
  type ReplaceEDITemplateSegmentsRequest,
  type SaveEDITestCaseRequest,
  ediShipmentLinkListSchema,
  ediShipmentLinkSchema,
  type RejectEDITransferRequest,
  type SubmitLoadTenderRequest,
  type UpdateEDITemplateRequest,
  type UpdateEDITemplateVersionRequest,
  ediTransferChangeListSchema,
  ediTransferChangeSchema,
  ediX12InspectionSchema,
  type UpsertEDICommunicationProfileRequest,
  type UpsertEDIPartnerRequest,
  type UpsertEDIPartnerDocumentProfileRequest,
} from "@/types/edi";
import { ediTransferEndpoints } from "./transfers";

export class EDIService {
  public async listConnections(query = "") {
    const response = await api.get(`/edi/connections/${query}`);
    return safeParse(ediConnectionListSchema, response, "EDIConnectionList");
  }

  public async createConnection(request: CreateEDIConnectionRequest) {
    const response = await api.post("/edi/connections/", request);
    return safeParse(ediConnectionSchema, response, "EDIConnection");
  }

  public async acceptConnection(connectionId: string) {
    const response = await api.post(`/edi/connections/${connectionId}/accept/`);
    return safeParse(ediConnectionSchema, response, "EDIConnection");
  }

  public async rejectConnection(connectionId: string, request: EDIConnectionActionRequest) {
    const response = await api.post(`/edi/connections/${connectionId}/reject/`, request);
    return safeParse(ediConnectionSchema, response, "EDIConnection");
  }

  public async suspendConnection(connectionId: string) {
    const response = await api.post(`/edi/connections/${connectionId}/suspend/`);
    return safeParse(ediConnectionSchema, response, "EDIConnection");
  }

  public async revokeConnection(connectionId: string) {
    const response = await api.post(`/edi/connections/${connectionId}/revoke/`);
    return safeParse(ediConnectionSchema, response, "EDIConnection");
  }

  public async createCommunicationProfile(request: UpsertEDICommunicationProfileRequest) {
    const response = await api.post("/edi/communication-profiles/", request);
    return safeParse(ediCommunicationProfileSchema, response, "EDICommunicationProfile");
  }

  public async updateCommunicationProfile(
    profileId: string,
    request: UpsertEDICommunicationProfileRequest,
  ) {
    const response = await api.put(`/edi/communication-profiles/${profileId}/`, request);
    return safeParse(ediCommunicationProfileSchema, response, "EDICommunicationProfile");
  }

  public async createTemplate(request: CreateEDITemplateRequest) {
    const response = await api.post("/edi/templates/", request);
    return safeParse(ediTemplateSchema, response, "EDITemplate");
  }

  public async getTemplate(templateId: string) {
    const response = await api.get(`/edi/templates/${templateId}/`);
    return safeParse(ediTemplateSchema, response, "EDITemplate");
  }

  public async updateTemplate(templateId: string, request: UpdateEDITemplateRequest) {
    const response = await api.put(`/edi/templates/${templateId}/`, request);
    return safeParse(ediTemplateSchema, response, "EDITemplate");
  }

  public async createTemplateDraft(templateId: string, request: CreateEDITemplateDraftRequest) {
    const response = await api.post(`/edi/templates/${templateId}/draft/`, request);
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async listTemplateVersions(templateId: string) {
    const response = await api.get(`/edi/templates/${templateId}/versions/`);
    return safeParse(ediTemplateVersionSchema.array(), response, "EDITemplateVersions");
  }

  public async getTemplateVersion(templateId: string, versionId: string) {
    const response = await api.get(`/edi/templates/${templateId}/versions/${versionId}/`);
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async updateTemplateVersion(
    templateId: string,
    versionId: string,
    request: UpdateEDITemplateVersionRequest,
  ) {
    const response = await api.put(`/edi/templates/${templateId}/versions/${versionId}/`, request);
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async replaceTemplateSegments(
    templateId: string,
    versionId: string,
    request: ReplaceEDITemplateSegmentsRequest,
  ) {
    const response = await api.put(
      `/edi/templates/${templateId}/versions/${versionId}/segments/`,
      request,
    );
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async replaceTemplateScriptLibraries(
    templateId: string,
    versionId: string,
    request: ReplaceEDITemplateScriptLibrariesRequest,
  ) {
    const response = await api.put(
      `/edi/templates/${templateId}/versions/${versionId}/script-libraries/`,
      request,
    );
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async validateTemplateVersion(templateId: string, versionId: string) {
    const response = await api.post(`/edi/templates/${templateId}/versions/${versionId}/validate/`);
    return safeParse(ediTemplateValidationResponseSchema, response, "EDITemplateValidation");
  }

  public async certifyTemplateVersion(
    templateId: string,
    versionId: string,
    request: EDITemplateActionRequest,
  ) {
    const response = await api.post(
      `/edi/templates/${templateId}/versions/${versionId}/certify/`,
      request,
    );
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async activateTemplateVersion(
    templateId: string,
    versionId: string,
    request: EDITemplateActionRequest,
  ) {
    const response = await api.post(
      `/edi/templates/${templateId}/versions/${versionId}/activate/`,
      request,
    );
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async archiveTemplateVersion(
    templateId: string,
    versionId: string,
    request: EDITemplateActionRequest,
  ) {
    const response = await api.post(
      `/edi/templates/${templateId}/versions/${versionId}/archive/`,
      request,
    );
    return safeParse(ediTemplateVersionSchema, response, "EDITemplateVersion");
  }

  public async listPartnerDocumentProfiles(query = "") {
    const response = await api.get(`/edi/document-profiles/${query}`);
    return safeParse(
      ediPartnerDocumentProfileListSchema,
      response,
      "EDIPartnerDocumentProfileList",
    );
  }

  public async createPartnerDocumentProfile(request: UpsertEDIPartnerDocumentProfileRequest) {
    const response = await api.post("/edi/document-profiles/", request);
    return safeParse(ediPartnerDocumentProfileSchema, response, "EDIPartnerDocumentProfile");
  }

  public async updatePartnerDocumentProfile(
    profileId: string,
    request: UpsertEDIPartnerDocumentProfileRequest,
  ) {
    const response = await api.put(`/edi/document-profiles/${profileId}/`, request);
    return safeParse(ediPartnerDocumentProfileSchema, response, "EDIPartnerDocumentProfile");
  }

  public async previewDocument(request: PreviewEDIDocumentRequest) {
    const response = await api.post("/edi/documents/preview/", request);
    return safeParse(ediDocumentPreviewSchema, response, "EDIDocumentPreview");
  }

  public async generateDocument(request: GenerateEDIDocumentRequest) {
    const response = await api.post("/edi/documents/generate/", request);
    return safeParse(ediMessageSchema, response, "EDIMessage");
  }

  public async listMessages(query = "") {
    const response = await api.get(`/edi/messages/${query}`);
    return safeParse(ediMessageListSchema, response, "EDIMessageList");
  }

  public async getMessage(messageId: string) {
    const response = await api.get(`/edi/messages/${messageId}/`);
    return safeParse(ediMessageSchema, response, "EDIMessage");
  }

  public async inspectMessage(messageId: string) {
    const response = await api.get(`/edi/messages/${messageId}/inspect/`);
    return safeParse(ediMessageInspectionSchema, response, "EDIMessageInspection");
  }

  public async testProfileConnection(profileId: string) {
    const response = await api.post(`/edi/communication-profiles/${profileId}/test-connection/`);
    return safeParse(ediConnectionTestResultSchema, response, "EDIConnectionTestResult");
  }

  public async inspectCertificate(certificate: string) {
    const response = await api.post(`/edi/communication-profiles/inspect-certificate/`, {
      certificate,
    });
    return safeParse(ediCertificateSummarySchema, response, "EDICertificateSummary");
  }

  public async bulkRetryMessageDelivery(messageIds: string[]) {
    const response = await api.post(`/edi/messages/bulk-retry-delivery/`, { messageIds });
    return safeParse(ediBulkActionResultSchema, response, "EDIBulkActionResult");
  }

  public async bulkReprocessInboundFiles(fileIds: string[]) {
    const response = await api.post(`/edi/inbound-files/bulk-reprocess/`, { fileIds });
    return safeParse(ediBulkActionResultSchema, response, "EDIBulkActionResult");
  }

  public async bulkApproveTransfers(transferIds: string[]) {
    const response = await api.post(`/edi/transfers/bulk-approve/`, { transferIds });
    return safeParse(ediBulkActionResultSchema, response, "EDIBulkActionResult");
  }

  public async bulkRejectTransfers(transferIds: string[], reason: string) {
    const response = await api.post(`/edi/transfers/bulk-reject/`, { transferIds, reason });
    return safeParse(ediBulkActionResultSchema, response, "EDIBulkActionResult");
  }

  public async retryMessageDelivery(messageId: string) {
    const response = await api.post(`/edi/messages/${messageId}/retry-delivery/`);
    return safeParse(ediMessageSchema, response, "EDIMessage");
  }

  public async getInboundFile(fileId: string) {
    const response = await api.get(`/edi/inbound-files/${fileId}/`);
    return safeParse(ediInboundFileSchema, response, "EDIInboundFile");
  }

  public async reprocessInboundFile(fileId: string) {
    const response = await api.post(`/edi/inbound-files/${fileId}/reprocess/`);
    return safeParse(ediInboundFileSchema, response, "EDIInboundFile");
  }

  public async inspectX12(request: InspectX12Request) {
    const response = await api.post("/edi/x12/inspect/", request);
    return safeParse(ediX12InspectionSchema, response, "EDIX12Inspection");
  }

  public async listTestCases(query = "") {
    const response = await api.get(`/edi/test-cases/${query}`);
    return safeParse(ediTestCaseListSchema, response, "EDITestCaseList");
  }

  public async getPartnerReadiness(partnerId: string) {
    const response = await api.get(`/edi/partners/${partnerId}/readiness/`);
    return safeParse(ediPartnerReadinessSchema, response, "EDIPartnerReadiness");
  }

  public async getTestCase(testCaseId: string) {
    const response = await api.get(`/edi/test-cases/${testCaseId}/`);
    return safeParse(ediTestCaseSchema, response, "EDITestCase");
  }

  public async createTestCase(request: SaveEDITestCaseRequest) {
    const response = await api.post("/edi/test-cases/", request);
    return safeParse(ediTestCaseSchema, response, "EDITestCase");
  }

  public async updateTestCase(testCaseId: string, request: SaveEDITestCaseRequest) {
    const response = await api.put(`/edi/test-cases/${testCaseId}/`, request);
    return safeParse(ediTestCaseSchema, response, "EDITestCase");
  }

  public async deleteTestCase(testCaseId: string) {
    return api.delete<undefined>(`/edi/test-cases/${testCaseId}/`);
  }

  public async previewTestCase(testCaseId: string) {
    const response = await api.post(`/edi/test-cases/${testCaseId}/preview/`);
    return safeParse(ediDocumentPreviewSchema, response, "EDIDocumentPreview");
  }

  public async createPartner(request: UpsertEDIPartnerRequest) {
    const response = await api.post("/edi/partners/", request);
    return safeParse(ediPartnerSchema, response, "EDIPartner");
  }

  public async createInternalPair(request: CreateInternalPartnerPairRequest) {
    const response = await api.post("/edi/partners/internal-pairs/", request);
    return safeParse(internalPartnerPairSchema, response, "InternalPartnerPair");
  }

  public async updatePartner(partnerId: string, request: UpsertEDIPartnerRequest) {
    const response = await api.put(`/edi/partners/${partnerId}/`, request);
    return safeParse(ediPartnerSchema, response, "EDIPartner");
  }

  public async getMappingProfile(partnerId: string) {
    const response = await api.get(`/edi/partners/${partnerId}/mapping-profile/`);
    return safeParse(ediMappingProfileSchema, response, "EDIMappingProfile");
  }

  public async saveMappingProfileItems(profileId: string, items: EDIMappingProfileItem[]) {
    const response = await api.put(`/edi/mapping-profiles/${profileId}/items/`, { items });
    return safeParse(ediMappingProfileItemSchema.array(), response, "EDIMappingProfileItems");
  }

  public async deleteMappingProfileItem(profileId: string, itemId: string) {
    return api.delete<undefined>(`/edi/mapping-profiles/${profileId}/items/${itemId}/`);
  }

  public async saveMappingProfile(partnerId: string, items: EDIMappingProfileItem[]) {
    const response = await api.put(`/edi/partners/${partnerId}/mapping-profile/`, { items });
    return safeParse(ediMappingProfileItemSchema.array(), response, "EDIMappingProfileItems");
  }

  public async deleteMappingItem(partnerId: string, itemId: string) {
    return api.delete<undefined>(`/edi/partners/${partnerId}/mapping-profile/items/${itemId}/`);
  }

  public async submitLoadTender(request: SubmitLoadTenderRequest) {
    const response = await api.post(ediTransferEndpoints.loadTenders, request);
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  public async getTransfer(transferId: string) {
    const response = await api.get(`/edi/transfers/${transferId}/`);
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  public async getMappingPreview(transferId: string) {
    const response = await api.get(`/edi/transfers/${transferId}/mapping-preview/`);
    return safeParse(ediMappingPreviewSchema, response, "EDIMappingPreview");
  }

  public async approveTransfer(transferId: string, request: ApproveEDITransferRequest) {
    const response = await api.post(`/edi/transfers/${transferId}/approve/`, request);
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  public async rejectTransfer(transferId: string, request: RejectEDITransferRequest) {
    const response = await api.post(`/edi/transfers/${transferId}/reject/`, request);
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  public async cancelTransfer(transferId: string) {
    const response = await api.post(`/edi/transfers/${transferId}/cancel/`);
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  public async expireTransfer(transferId: string) {
    const response = await api.post(`/edi/transfers/${transferId}/expire/`);
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  public async listShipmentLinks(query: string) {
    const response = await api.get(`/edi/shipment-links/${query}`);
    return safeParse(ediShipmentLinkListSchema, response, "EDIShipmentLinkList");
  }

  public async getShipmentLink(linkId: string) {
    const response = await api.get(`/edi/shipment-links/${linkId}/`);
    return safeParse(ediShipmentLinkSchema, response, "EDIShipmentLink");
  }

  public async listTransferChanges(query: string) {
    const response = await api.get(`/edi/transfer-changes/${query}`);
    return safeParse(ediTransferChangeListSchema, response, "EDITransferChangeList");
  }

  public async getTransferChange(changeId: string) {
    const response = await api.get(`/edi/transfer-changes/${changeId}/`);
    return safeParse(ediTransferChangeSchema, response, "EDITransferChange");
  }

  public async applyTransferChange(changeId: string, reason?: string) {
    const response = await api.post(`/edi/transfer-changes/${changeId}/apply/`, { reason });
    return safeParse(ediTransferChangeSchema, response, "EDITransferChange");
  }

  public async rejectTransferChange(changeId: string, reason?: string) {
    const response = await api.post(`/edi/transfer-changes/${changeId}/reject/`, { reason });
    return safeParse(ediTransferChangeSchema, response, "EDITransferChange");
  }
}
