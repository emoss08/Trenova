import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  ediMappingPreviewSchema,
  ediMappingProfileListSchema,
  ediMappingProfileItemSchema,
  ediMappingProfileSchema,
  ediCommunicationProfileListSchema,
  ediCommunicationProfileSchema,
  ediConnectionListSchema,
  ediConnectionSchema,
  ediDocumentPreviewSchema,
  ediDocumentTypeSchema,
  ediMessageListSchema,
  ediMessageSchema,
  ediPartnerDocumentProfileListSchema,
  ediPartnerDocumentProfileSchema,
  ediPartnerListSchema,
  ediPartnerSchema,
  ediPartnerSelectOptionListSchema,
  ediTemplateListSchema,
  ediTestCaseListSchema,
  ediTransferListSchema,
  ediTransferSchema,
  internalPartnerPairSchema,
  type ApproveEDITransferRequest,
  type CreateEDIConnectionRequest,
  type CreateInternalPartnerPairRequest,
  type EDIConnectionActionRequest,
  type EDIMappingProfileItem,
  type EDIPartner,
  type GenerateEDIDocumentRequest,
  type PreviewEDIDocumentRequest,
  ediShipmentLinkListSchema,
  ediShipmentLinkSchema,
  type RejectEDITransferRequest,
  type SubmitLoadTenderRequest,
  ediTransferChangeListSchema,
  ediTransferChangeSchema,
  type UpsertEDICommunicationProfileRequest,
  type UpsertEDIPartnerDocumentProfileRequest,
} from "@/types/edi";

export class EDIService {
  public async listPartners(query = "") {
    const response = await api.get(`/edi/partners/${query}`);
    return safeParse(ediPartnerListSchema, response, "EDIPartnerList");
  }

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

  public async listCommunicationProfiles(query = "") {
    const response = await api.get(`/edi/communication-profiles/${query}`);
    return safeParse(ediCommunicationProfileListSchema, response, "EDICommunicationProfileList");
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

  public async listDocumentTypes() {
    const response = await api.get("/edi/document-types/?standard=X12&transactionSet=204&direction=Outbound");
    return safeParse(ediDocumentTypeSchema.array(), response, "EDIDocumentTypes");
  }

  public async listTemplates(query = "") {
    const response = await api.get(`/edi/templates/${query}`);
    return safeParse(ediTemplateListSchema, response, "EDITemplateList");
  }

  public async listPartnerDocumentProfiles(query = "") {
    const response = await api.get(`/edi/document-profiles/${query}`);
    return safeParse(ediPartnerDocumentProfileListSchema, response, "EDIPartnerDocumentProfileList");
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

  public async listTestCases(query = "") {
    const response = await api.get(`/edi/test-cases/${query}`);
    return safeParse(ediTestCaseListSchema, response, "EDITestCaseList");
  }

  public async previewTestCase(testCaseId: string) {
    const response = await api.post(`/edi/test-cases/${testCaseId}/preview/`);
    return safeParse(ediDocumentPreviewSchema, response, "EDITestCasePreview");
  }

  public async listInboundTransfers(query = "") {
    const response = await api.get(`/edi/transfers/inbound/${query}`);
    return safeParse(ediTransferListSchema, response, "EDIInboundTransferList");
  }

  public async listOutboundTransfers(query = "") {
    const response = await api.get(`/edi/transfers/outbound/${query}`);
    return safeParse(ediTransferListSchema, response, "EDIOutboundTransferList");
  }

  public async selectPartners(kind = "Internal") {
    const response = await api.get(`/edi/partners/select-options/?kind=${kind}&limit=100`);
    return safeParse(ediPartnerSelectOptionListSchema, response, "EDIPartnerOptions");
  }

  public async createInternalPair(request: CreateInternalPartnerPairRequest) {
    const response = await api.post("/edi/partners/internal-pairs/", request);
    return safeParse(internalPartnerPairSchema, response, "InternalPartnerPair");
  }

  public async updatePartner(partner: EDIPartner) {
    const response = await api.put(`/edi/partners/${partner.id}/`, partner);
    return safeParse(ediPartnerSchema, response, "EDIPartner");
  }

  public async getMappingProfile(partnerId: string) {
    const response = await api.get(`/edi/partners/${partnerId}/mapping-profile/`);
    return safeParse(ediMappingProfileSchema, response, "EDIMappingProfile");
  }

  public async listMappingProfiles(query = "") {
    const response = await api.get(`/edi/mapping-profiles/${query}`);
    return safeParse(ediMappingProfileListSchema, response, "EDIMappingProfileList");
  }

  public async getMappingProfileById(profileId: string) {
    const response = await api.get(`/edi/mapping-profiles/${profileId}/`);
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
    const response = await api.post("/edi/transfers/load-tenders/", request);
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
