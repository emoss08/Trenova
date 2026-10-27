import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  ediBulkActionResultSchema,
  ediPartnerReadinessSchema,
  ediCertificateSummarySchema,
  ediConnectionTestResultSchema,
  ediPollResultSchema,
  ediMappingPreviewSchema,
  ediMappingProfileItemSchema,
  ediMappingProfileSchema,
  ediCommunicationProfileSchema,
  ediConnectionListSchema,
  ediConnectionSchema,
  ediDocumentPreviewSchema,
  ediMessageInspectionSchema,
  ediInboundFileInspectionSchema,
  ediInboundFileSchema,
  ediMessageListSchema,
  ediMessageSchema,
  ediPartnerSchema,
  ediTransferListSchema,
  ediTransferSchema,
  type EDITransfer,
  internalPartnerPairSchema,
  type ApproveEDITransferRequest,
  type CreateEDIConnectionRequest,
  type CreateInternalPartnerPairRequest,
  type EDIConnectionActionRequest,
  type EDIMappingProfileItem,
  type GenerateEDIDocumentRequest,
  type InspectX12Request,
  type PreviewEDIDocumentRequest,
  ediShipmentLinkListSchema,
  ediShipmentLinkSchema,
  type RejectEDITransferRequest,
  type SubmitLoadTenderRequest,
  ediTransferChangeListSchema,
  ediTransferChangeSchema,
  ediX12InspectionSchema,
  type UpsertEDICommunicationProfileRequest,
  type UpsertEDIPartnerRequest,
} from "@trenova/shared/types/edi";
import { ediTransferEndpoints } from "./transfers";

/** The most tenders one inbound file is listed with: the transfer list's page limit. */
const INBOUND_FILE_TRANSFER_LIMIT = 100;

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

  public async getMessage(messageId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/messages/${messageId}/`, { signal: options?.signal });
    return safeParse(ediMessageSchema, response, "EDIMessage");
  }

  public async inspectMessage(messageId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/messages/${messageId}/inspect/`, {
      signal: options?.signal,
    });
    return safeParse(ediMessageInspectionSchema, response, "EDIMessageInspection");
  }

  public async testProfileConnection(profileId: string) {
    const response = await api.post(`/edi/communication-profiles/${profileId}/test-connection/`);
    return safeParse(ediConnectionTestResultSchema, response, "EDIConnectionTestResult");
  }

  public async pollProfile(profileId: string) {
    const response = await api.post(`/edi/communication-profiles/${profileId}/poll/`);
    return safeParse(ediPollResultSchema, response, "EDIPollResult");
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

  public async replayMessageDelivery(messageId: string) {
    const response = await api.post(`/edi/messages/${messageId}/replay/`);
    return safeParse(ediMessageSchema, response, "EDIMessage");
  }

  public async retryMessageDelivery(messageId: string) {
    const response = await api.post(`/edi/messages/${messageId}/retry-delivery/`);
    return safeParse(ediMessageSchema, response, "EDIMessage");
  }

  public async getInboundFile(fileId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/inbound-files/${fileId}/`, { signal: options?.signal });
    return safeParse(ediInboundFileSchema, response, "EDIInboundFile");
  }

  public async inspectInboundFile(fileId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/inbound-files/${fileId}/inspect/`, {
      signal: options?.signal,
    });
    return safeParse(ediInboundFileInspectionSchema, response, "EDIInboundFileInspection");
  }

  public async reprocessInboundFile(fileId: string) {
    const response = await api.post(`/edi/inbound-files/${fileId}/reprocess/`);
    return safeParse(ediInboundFileSchema, response, "EDIInboundFile");
  }

  public async inspectX12(request: InspectX12Request) {
    const response = await api.post("/edi/x12/inspect/", request);
    return safeParse(ediX12InspectionSchema, response, "EDIX12Inspection");
  }

  public async getPartnerReadiness(partnerId: string) {
    const response = await api.get(`/edi/partners/${partnerId}/readiness/`);
    return safeParse(ediPartnerReadinessSchema, response, "EDIPartnerReadiness");
  }

  public async getPartner(partnerId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/partners/${partnerId}/`, { signal: options?.signal });
    return safeParse(ediPartnerSchema, response, "EDIPartner");
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

  public async getTransfer(transferId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/transfers/${transferId}/`, { signal: options?.signal });
    return safeParse(ediTransferSchema, response, "EDITransfer");
  }

  /**
   * Every tender one inbound file's 204s opened. A batch can carry more than a
   * page, so each page's cursor reads the next until the list says it is done:
   * its total can be an estimate, which says nothing about where the list ends.
   */
  public async listInboundFileTransfers(fileId: string, options?: { signal?: AbortSignal }) {
    const page = async (after: string | null) => {
      const query = new URLSearchParams({
        direction: "inbound",
        inboundFileId: fileId,
        limit: String(INBOUND_FILE_TRANSFER_LIMIT),
      });
      if (after) query.set("after", after);
      const response = await api.get(`/edi/transfers/?${query.toString()}`, {
        signal: options?.signal,
      });
      return safeParse(ediTransferListSchema, response, "EDITransferList");
    };

    const transfers: EDITransfer[] = [];
    let after: string | null = null;
    do {
      const list = await page(after);
      transfers.push(...list.results);
      after = list.hasNextPage && list.endCursor ? list.endCursor : null;
    } while (after !== null);

    return transfers;
  }

  public async getMappingPreview(transferId: string, options?: { signal?: AbortSignal }) {
    const response = await api.get(`/edi/transfers/${transferId}/mapping-preview/`, {
      signal: options?.signal,
    });
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
