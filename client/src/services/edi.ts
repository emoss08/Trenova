import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  ediMappingPreviewSchema,
  ediMappingProfileItemSchema,
  ediMappingProfileSchema,
  ediPartnerListSchema,
  ediPartnerSchema,
  ediPartnerSelectOptionListSchema,
  ediTransferListSchema,
  ediTransferSchema,
  internalPartnerPairSchema,
  type ApproveEDITransferRequest,
  type CreateInternalPartnerPairRequest,
  type EDIMappingProfileItem,
  type EDIPartner,
  type RejectEDITransferRequest,
  type SubmitLoadTenderRequest,
} from "@/types/edi";

export class EDIService {
  public async listPartners(query = "") {
    const response = await api.get(`/edi/partners/${query}`);
    return safeParse(ediPartnerListSchema, response, "EDIPartnerList");
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
}
