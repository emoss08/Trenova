import {
  AcceptEdiConnectionDocument,
  ApproveEdiTransferDocument,
  BulkApproveEdiTransfersDocument,
  BulkRejectEdiTransfersDocument,
  BulkReprocessEdiInboundFilesDocument,
  BulkRetryEdiMessageDeliveryDocument,
  CancelEdiTransferDocument,
  CreateEdiCommunicationProfileDocument,
  CreateEdiConnectionDocument,
  CreateEdiPartnerDocument,
  DeleteEdiMappingProfileItemDocument,
  DeleteEdiPartnerMappingItemDocument,
  EdiInboundFileInspectionDocument,
  EdiInboundFileRecordDocument,
  EdiInboundFileTransfersDocument,
  EdiMessageInspectionDocument,
  EdiMessageRecordDocument,
  EdiPartnerMappingProfileDocument,
  EdiPartnerReadinessDetailDocument,
  EdiPartnerRecordDocument,
  EdiTransferMappingPreviewDocument,
  EdiTransferRecordDocument,
  InspectEdiCertificateDocument,
  PendingEdiConnectionsDocument,
  PollEdiCommunicationProfileDocument,
  RejectEdiConnectionDocument,
  RejectEdiTransferDocument,
  ReplayEdiMessageDocument,
  ReprocessEdiInboundFileDocument,
  RetryEdiMessageDeliveryDocument,
  SaveEdiMappingProfileItemsDocument,
  SaveEdiPartnerMappingProfileDocument,
  SubmitEdiLoadTenderDocument,
  TestEdiCommunicationProfileConnectionDocument,
  UpdateEdiCommunicationProfileDocument,
  UpdateEdiPartnerDocument,
  EdiPartnerRecordFieldsFragmentDoc,
  type EdiInboundFileTransfersQuery,
  type EntityStatus,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  ediConnectionTestResultSchema,
  ediInboundFileInspectionSchema,
  ediMappingPreviewSchema,
  ediMappingProfileSchema,
  type ApproveEDITransferRequest,
  type CreateEDIConnectionRequest,
  type EDIConnectionActionRequest,
  type EDIMappingProfileItem,
  type EDITransfer,
  type RejectEDITransferRequest,
  type SubmitLoadTenderRequest,
  type UpsertEDICommunicationProfileRequest,
  type UpsertEDIPartnerRequest,
} from "@trenova/shared/types/edi";
import {
  toEDIInboundFile,
  toEDIMessage,
  toEDIMessageInspection,
  toEDITransfer,
  toMappingItemInputs,
} from "@/lib/graphql/edi-records";

type Options = { signal?: AbortSignal };

/** The most tenders one inbound file is listed with per page: the transfer list's page limit. */
const INBOUND_FILE_TRANSFER_LIMIT = 100;

/** How many connections waiting on this organization the pending panel lists. */
const PENDING_CONNECTION_LIMIT = 25;

export class EDIService {
  public async listPendingConnections(options?: Options) {
    const response = await requestGraphQL({
      document: PendingEdiConnectionsDocument,
      operationName: "PendingEdiConnections",
      variables: { limit: PENDING_CONNECTION_LIMIT },
      signal: options?.signal,
    });
    return response.pendingEdiConnections;
  }

  public async createConnection(request: CreateEDIConnectionRequest) {
    const response = await requestGraphQL({
      document: CreateEdiConnectionDocument,
      operationName: "CreateEdiConnection",
      variables: { input: request },
    });
    return response.createEdiConnection;
  }

  public async acceptConnection(connectionId: string) {
    const response = await requestGraphQL({
      document: AcceptEdiConnectionDocument,
      operationName: "AcceptEdiConnection",
      variables: { id: connectionId },
    });
    return response.acceptEdiConnection;
  }

  public async rejectConnection(connectionId: string, request: EDIConnectionActionRequest) {
    const response = await requestGraphQL({
      document: RejectEdiConnectionDocument,
      operationName: "RejectEdiConnection",
      variables: { id: connectionId, reason: request.reason },
    });
    return response.rejectEdiConnection;
  }

  public async createCommunicationProfile(request: UpsertEDICommunicationProfileRequest) {
    const response = await requestGraphQL({
      document: CreateEdiCommunicationProfileDocument,
      operationName: "CreateEdiCommunicationProfile",
      variables: { input: toCommunicationProfileInput(request) },
    });
    return response.createEdiCommunicationProfile;
  }

  public async updateCommunicationProfile(
    profileId: string,
    request: UpsertEDICommunicationProfileRequest,
  ) {
    const response = await requestGraphQL({
      document: UpdateEdiCommunicationProfileDocument,
      operationName: "UpdateEdiCommunicationProfile",
      variables: { id: profileId, input: toCommunicationProfileInput(request) },
    });
    return response.updateEdiCommunicationProfile;
  }

  public async getMessage(messageId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiMessageRecordDocument,
      operationName: "EdiMessageRecord",
      variables: { id: messageId },
      signal: options?.signal,
    });
    const detail = response.ediMessage;
    return toEDIMessage(detail.message, {
      rawX12: detail.message.rawX12,
      diagnosticCount: detail.diagnosticCount,
      diagnostics: detail.diagnostics,
    });
  }

  public async inspectMessage(messageId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiMessageInspectionDocument,
      operationName: "EdiMessageInspection",
      variables: { id: messageId },
      signal: options?.signal,
    });
    return toEDIMessageInspection(response.ediMessageInspection);
  }

  public async testProfileConnection(profileId: string) {
    const response = await requestGraphQL({
      document: TestEdiCommunicationProfileConnectionDocument,
      operationName: "TestEdiCommunicationProfileConnection",
      variables: { id: profileId },
    });
    return safeParse(
      ediConnectionTestResultSchema,
      response.testEdiCommunicationProfileConnection,
      "EDIConnectionTestResult",
    );
  }

  public async pollProfile(profileId: string) {
    const response = await requestGraphQL({
      document: PollEdiCommunicationProfileDocument,
      operationName: "PollEdiCommunicationProfile",
      variables: { id: profileId },
    });
    return response.pollEdiCommunicationProfile;
  }

  public async inspectCertificate(certificate: string, options?: Options) {
    const response = await requestGraphQL({
      document: InspectEdiCertificateDocument,
      operationName: "InspectEdiCertificate",
      variables: { certificate },
      signal: options?.signal,
    });
    return response.inspectEdiCertificate;
  }

  public async bulkRetryMessageDelivery(messageIds: string[]) {
    const response = await requestGraphQL({
      document: BulkRetryEdiMessageDeliveryDocument,
      operationName: "BulkRetryEdiMessageDelivery",
      variables: { ids: messageIds },
    });
    return response.bulkRetryEdiMessageDelivery;
  }

  public async bulkReprocessInboundFiles(fileIds: string[]) {
    const response = await requestGraphQL({
      document: BulkReprocessEdiInboundFilesDocument,
      operationName: "BulkReprocessEdiInboundFiles",
      variables: { ids: fileIds },
    });
    return response.bulkReprocessEdiInboundFiles;
  }

  public async bulkApproveTransfers(transferIds: string[]) {
    const response = await requestGraphQL({
      document: BulkApproveEdiTransfersDocument,
      operationName: "BulkApproveEdiTransfers",
      variables: { ids: transferIds },
    });
    return response.bulkApproveEdiTransfers;
  }

  public async bulkRejectTransfers(transferIds: string[], reason: string) {
    const response = await requestGraphQL({
      document: BulkRejectEdiTransfersDocument,
      operationName: "BulkRejectEdiTransfers",
      variables: { ids: transferIds, reason },
    });
    return response.bulkRejectEdiTransfers;
  }

  public async replayMessageDelivery(messageId: string) {
    const response = await requestGraphQL({
      document: ReplayEdiMessageDocument,
      operationName: "ReplayEdiMessage",
      variables: { id: messageId },
    });
    return response.replayEdiMessage;
  }

  public async retryMessageDelivery(messageId: string) {
    const response = await requestGraphQL({
      document: RetryEdiMessageDeliveryDocument,
      operationName: "RetryEdiMessageDelivery",
      variables: { id: messageId },
    });
    return response.retryEdiMessageDelivery;
  }

  public async getInboundFile(fileId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiInboundFileRecordDocument,
      operationName: "EdiInboundFileRecord",
      variables: { id: fileId },
      signal: options?.signal,
    });
    return toEDIInboundFile(response.ediInboundFile);
  }

  public async inspectInboundFile(fileId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiInboundFileInspectionDocument,
      operationName: "EdiInboundFileInspection",
      variables: { id: fileId },
      signal: options?.signal,
    });
    return safeParse(
      ediInboundFileInspectionSchema,
      response.ediInboundFileInspection,
      "EDIInboundFileInspection",
    );
  }

  public async reprocessInboundFile(fileId: string) {
    const response = await requestGraphQL({
      document: ReprocessEdiInboundFileDocument,
      operationName: "ReprocessEdiInboundFile",
      variables: { id: fileId },
    });
    return response.reprocessEdiInboundFile;
  }

  public async getPartnerReadiness(partnerId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiPartnerReadinessDetailDocument,
      operationName: "EdiPartnerReadinessDetail",
      variables: { partnerIds: [partnerId] },
      signal: options?.signal,
    });
    return response.ediPartnerReadiness[0] ?? null;
  }

  public async getPartner(partnerId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiPartnerRecordDocument,
      operationName: "EdiPartnerRecord",
      variables: { id: partnerId },
      signal: options?.signal,
    });
    return getFragmentData(EdiPartnerRecordFieldsFragmentDoc, response.ediPartner);
  }

  public async createPartner(request: UpsertEDIPartnerRequest) {
    const response = await requestGraphQL({
      document: CreateEdiPartnerDocument,
      operationName: "CreateEdiPartner",
      variables: { input: toPartnerInput(request) },
    });
    return response.createEdiPartner;
  }

  public async updatePartner(partnerId: string, request: UpsertEDIPartnerRequest) {
    const response = await requestGraphQL({
      document: UpdateEdiPartnerDocument,
      operationName: "UpdateEdiPartner",
      variables: { id: partnerId, input: toPartnerInput(request) },
    });
    return response.updateEdiPartner;
  }

  public async getMappingProfile(partnerId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiPartnerMappingProfileDocument,
      operationName: "EdiPartnerMappingProfile",
      variables: { partnerId },
      signal: options?.signal,
    });
    return safeParse(
      ediMappingProfileSchema,
      response.ediPartnerMappingProfile,
      "EDIMappingProfile",
    );
  }

  public async saveMappingProfileItems(profileId: string, items: EDIMappingProfileItem[]) {
    const response = await requestGraphQL({
      document: SaveEdiMappingProfileItemsDocument,
      operationName: "SaveEdiMappingProfileItems",
      variables: { profileId, items: toMappingItemInputs(items) },
    });
    return response.saveEdiMappingProfileItems;
  }

  public async deleteMappingProfileItem(profileId: string, itemId: string) {
    await requestGraphQL({
      document: DeleteEdiMappingProfileItemDocument,
      operationName: "DeleteEdiMappingProfileItem",
      variables: { profileId, itemId },
    });
  }

  public async saveMappingProfile(partnerId: string, items: EDIMappingProfileItem[]) {
    const response = await requestGraphQL({
      document: SaveEdiPartnerMappingProfileDocument,
      operationName: "SaveEdiPartnerMappingProfile",
      variables: { partnerId, items: toMappingItemInputs(items) },
    });
    return response.saveEdiPartnerMappingProfile;
  }

  public async deleteMappingItem(partnerId: string, itemId: string) {
    await requestGraphQL({
      document: DeleteEdiPartnerMappingItemDocument,
      operationName: "DeleteEdiPartnerMappingItem",
      variables: { partnerId, itemId },
    });
  }

  public async submitLoadTender(request: SubmitLoadTenderRequest) {
    const response = await requestGraphQL({
      document: SubmitEdiLoadTenderDocument,
      operationName: "SubmitEdiLoadTender",
      variables: { input: request },
    });
    return response.submitEdiLoadTender;
  }

  public async getTransfer(transferId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiTransferRecordDocument,
      operationName: "EdiTransferRecord",
      variables: { id: transferId },
      signal: options?.signal,
    });
    return toEDITransfer(response.ediTransfer.transfer);
  }

  /**
   * Every tender one inbound file's 204s opened. A batch can carry more than a
   * page, so each page's cursor reads the next until the list says it is done:
   * its total can be an estimate, which says nothing about where the list ends.
   */
  public async listInboundFileTransfers(fileId: string, options?: Options) {
    const transfers: EDITransfer[] = [];
    let after: string | null = null;
    do {
      const response: EdiInboundFileTransfersQuery = await requestGraphQL({
        document: EdiInboundFileTransfersDocument,
        operationName: "EdiInboundFileTransfers",
        variables: {
          inboundFileId: fileId,
          input: { first: INBOUND_FILE_TRANSFER_LIMIT, after },
        },
        signal: options?.signal,
      });
      const page = response.ediTransfers;
      for (const edge of page.edges) {
        transfers.push(await toEDITransfer(edge.node));
      }
      after = page.pageInfo.hasNextPage && page.pageInfo.endCursor ? page.pageInfo.endCursor : null;
    } while (after !== null);

    return transfers;
  }

  public async getMappingPreview(transferId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiTransferMappingPreviewDocument,
      operationName: "EdiTransferMappingPreview",
      variables: { id: transferId },
      signal: options?.signal,
    });
    return safeParse(
      ediMappingPreviewSchema,
      response.ediTransferMappingPreview,
      "EDIMappingPreview",
    );
  }

  public async approveTransfer(transferId: string, request: ApproveEDITransferRequest) {
    const response = await requestGraphQL({
      document: ApproveEdiTransferDocument,
      operationName: "ApproveEdiTransfer",
      variables: { id: transferId, mappings: toMappingItemInputs(request.mappings) },
    });
    return response.approveEdiTransfer;
  }

  public async rejectTransfer(transferId: string, request: RejectEDITransferRequest) {
    const response = await requestGraphQL({
      document: RejectEdiTransferDocument,
      operationName: "RejectEdiTransfer",
      variables: { id: transferId, reason: request.reason },
    });
    return response.rejectEdiTransfer;
  }

  public async cancelTransfer(transferId: string) {
    const response = await requestGraphQL({
      document: CancelEdiTransferDocument,
      operationName: "CancelEdiTransfer",
      variables: { id: transferId },
    });
    return response.cancelEdiTransfer;
  }
}

function toPartnerInput(request: UpsertEDIPartnerRequest) {
  return { ...request, status: request.status as EntityStatus | undefined };
}

function toCommunicationProfileInput(request: UpsertEDICommunicationProfileRequest) {
  return { ...request, status: request.status as EntityStatus | undefined };
}
