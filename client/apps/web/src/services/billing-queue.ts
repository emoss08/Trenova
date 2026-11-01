import { z } from "zod";
import {
  AssignBillingQueueItemBillerDocument,
  BillingQueueAssistDocument,
  BillingQueueBriefDocument,
  BillingQueueBulkRunDocument,
  BillingQueueFilterPresetsDocument,
  BillingQueueItemActivityDocument,
  BillingQueueItemDetailDocument,
  BillingQueueNeighborsDocument,
  BillingQueueNovaGenerationDocument,
  BillingQueueRecommendationsDocument,
  BillingQueueSearchIntentDocument,
  BillingQueueStatsDocument,
  BillingQueueSummariesDocument,
  CancelBillingQueueBulkRunDocument,
  CreateBillingQueueFilterPresetDocument,
  DeleteBillingQueueFilterPresetDocument,
  PostBillingQueueItemDocument,
  ReassignBillingQueueChargeDocument,
  RedetectBillingQueuePodSignaturesDocument,
  ReleaseBillingQueueItemDocument,
  ResolveBillingQueueIssueDocument,
  StartBillingQueueBulkRunDocument,
  UndoBillingQueueIssueDocument,
  UpdateBillingQueueChargesDocument,
  UpdateBillingQueueFilterPresetDocument,
  UpdateBillingQueueInvoiceDetailsDocument,
  UpdateBillingQueueItemStatusDocument,
  type BillingQueueBulkAction,
  type BillingQueueHoldReasonCode,
  type BillingQueueStatus,
  type BillingQueueExceptionReasonCode,
  type ChargeAllocationKind,
  type ChargeAllocationMethod,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  billingQueueApprovalRunSchema,
  billingQueueAssistSchema,
  billingQueueBriefSchema,
  billingQueueNovaGenerationSchema,
  billingQueueRecommendationSchema,
  billingQueueSearchIntentSchema,
  billingQueueEventPageSchema,
  billingQueueFilterPresetSchema,
  billingQueueItemSchema,
  billingQueueNeighborsSchema,
  billingQueuePostResultSchema,
  billingQueueSummarySchema,
  billingQueueStatsSchema,
  podSignatureRedetectResultSchema,
  reassignChargeResultSchema,
  type BillingQueueAssignInput,
  type BillingQueueInvoiceDetailsInput,
  type BulkRunAction,
  type BulkRunParams,
  type BillingQueueFilterPresetInput,
  type BillingQueueUpdateChargesInput,
  type BillingQueueUpdateStatusInput,
  type ReassignChargeInput,
} from "@trenova/shared/types/billing-queue";

type RequestOptions = { signal?: AbortSignal };

export class BillingQueueService {
  /** Lane counts, plus the pulse of the caller's day when `dayStart` is given. */
  public async getStats(dayStart?: number, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueStats: unknown }>({
      document: BillingQueueStatsDocument,
      operationName: "BillingQueueStats",
      variables: { dayStart: dayStart || undefined },
      signal: options.signal,
    });
    return safeParse(billingQueueStatsSchema, data.billingQueueStats, "BillingQueueStats");
  }

  public async getRecommendations(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueRecommendations: unknown }>({
      document: BillingQueueRecommendationsDocument,
      operationName: "BillingQueueRecommendations",
      signal: options.signal,
    });
    return safeParse(
      z.array(billingQueueRecommendationSchema),
      data.billingQueueRecommendations ?? [],
      "BillingQueueRecommendations",
    );
  }

  public async getAssist(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueAssist: unknown }>({
      document: BillingQueueAssistDocument,
      operationName: "BillingQueueAssist",
      signal: options.signal,
    });
    return safeParse(billingQueueAssistSchema, data.billingQueueAssist, "BillingQueueAssist");
  }

  public async getBrief(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueBrief: unknown }>({
      document: BillingQueueBriefDocument,
      operationName: "BillingQueueBrief",
      signal: options.signal,
    });
    return safeParse(billingQueueBriefSchema, data.billingQueueBrief, "BillingQueueBrief");
  }

  /**
   * What Nova is generating for the whole queue. Reading it starts a generation
   * when the queue has items Nova has not reviewed or no brief.
   */
  public async getNovaGeneration(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueNovaGeneration: unknown }>({
      document: BillingQueueNovaGenerationDocument,
      operationName: "BillingQueueNovaGeneration",
      signal: options.signal,
    });
    return safeParse(
      billingQueueNovaGenerationSchema,
      data.billingQueueNovaGeneration ?? null,
      "BillingQueueNovaGeneration",
    );
  }

  /** Turns a sentence into queue filters; `today` (YYYY-MM-DD) is the person's own date. */
  public async composeSearch(text: string, today: string) {
    const data = await requestGraphQL<{ billingQueueSearchIntent: unknown }>({
      document: BillingQueueSearchIntentDocument,
      operationName: "BillingQueueSearchIntent",
      variables: { text, today },
    });
    return safeParse(
      billingQueueSearchIntentSchema,
      data.billingQueueSearchIntent,
      "BillingQueueSearchIntent",
    );
  }

  public async updateInvoiceDetails(id: string, payload: BillingQueueInvoiceDetailsInput) {
    const data = await requestGraphQL<{ updateBillingQueueInvoiceDetails: unknown }>({
      document: UpdateBillingQueueInvoiceDetailsDocument,
      operationName: "UpdateBillingQueueInvoiceDetails",
      variables: { id, input: payload },
    });
    return safeParse(
      billingQueueItemSchema,
      data.updateBillingQueueInvoiceDetails,
      "BillingQueueItem",
    );
  }

  /** The item with everything a biller reads to decide it. */
  public async getById(id: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueItem: unknown }>({
      document: BillingQueueItemDetailDocument,
      operationName: "BillingQueueItemDetail",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(billingQueueItemSchema, data.billingQueueItem, "BillingQueueItem");
  }

  public async updateStatus(id: string, payload: BillingQueueUpdateStatusInput) {
    const data = await requestGraphQL<{ updateBillingQueueStatus: unknown }>({
      document: UpdateBillingQueueItemStatusDocument,
      operationName: "UpdateBillingQueueItemStatus",
      variables: {
        id,
        input: {
          ...payload,
          status: payload.status as BillingQueueStatus,
          exceptionReasonCode: payload.exceptionReasonCode as
            | BillingQueueExceptionReasonCode
            | undefined,
          holdReasonCode: payload.holdReasonCode as BillingQueueHoldReasonCode | undefined,
        },
      },
    });
    return safeParse(billingQueueItemSchema, data.updateBillingQueueStatus, "BillingQueueItem");
  }

  public async assign(id: string, payload: BillingQueueAssignInput) {
    const data = await requestGraphQL<{ assignBillingQueueBiller: unknown }>({
      document: AssignBillingQueueItemBillerDocument,
      operationName: "AssignBillingQueueItemBiller",
      variables: { id, input: payload },
    });
    return safeParse(billingQueueItemSchema, data.assignBillingQueueBiller, "BillingQueueItem");
  }

  public async updateCharges(id: string, payload: BillingQueueUpdateChargesInput) {
    const data = await requestGraphQL<{ updateBillingQueueCharges: unknown }>({
      document: UpdateBillingQueueChargesDocument,
      operationName: "UpdateBillingQueueCharges",
      variables: {
        id,
        input: {
          formulaTemplateId: payload.formulaTemplateId,
          tariffInputs: payload.tariffInputs,
          convertAmountSplitsToPercent: payload.convertAmountSplitsToPercent,
          additionalCharges: payload.additionalCharges?.map((charge) => ({
            id: charge.id,
            accessorialChargeId: charge.accessorialChargeId,
            method: charge.method,
            amount: String(charge.amount),
            unit: charge.unit,
          })),
        },
      },
    });
    return safeParse(billingQueueItemSchema, data.updateBillingQueueCharges, "BillingQueueItem");
  }

  /**
   * Changes who pays for one charge on the item's shipment while it is in
   * review. The server creates or cancels sibling queue items as payers gain or
   * lose a share.
   */
  public async reassignCharge(id: string, payload: ReassignChargeInput) {
    const data = await requestGraphQL<{ reassignBillingQueueCharge: unknown }>({
      document: ReassignBillingQueueChargeDocument,
      operationName: "ReassignBillingQueueCharge",
      variables: {
        id,
        input: {
          chargeKind: payload.chargeKind as ChargeAllocationKind,
          additionalChargeId: payload.additionalChargeId,
          allocations: payload.allocations.map((allocation) => ({
            id: allocation.id,
            billToCustomerId: allocation.billToCustomerId,
            method: allocation.method as ChargeAllocationMethod,
            percent: allocation.percent,
            amount: allocation.amount,
          })),
        },
      },
    });
    return safeParse(
      reassignChargeResultSchema,
      data.reassignBillingQueueCharge,
      "ReassignChargeResult",
    );
  }

  /** The item's place in the queue: what comes before and after it, and where it sits. */
  public async getNeighbors(id: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueNeighbors: unknown }>({
      document: BillingQueueNeighborsDocument,
      operationName: "BillingQueueNeighbors",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(
      billingQueueNeighborsSchema,
      data.billingQueueNeighbors,
      "BillingQueueNeighbors",
    );
  }

  /** The item's activity, newest first, a page at a time. */
  public async getActivity(
    id: string,
    before?: { at: number; id: string },
    limit = 30,
    options: RequestOptions = {},
  ) {
    const data = await requestGraphQL<{ billingQueueItemActivity: unknown }>({
      document: BillingQueueItemActivityDocument,
      operationName: "BillingQueueItemActivity",
      variables: { id, limit, before },
      signal: options.signal,
    });
    return safeParse(
      billingQueueEventPageSchema,
      data.billingQueueItemActivity,
      "BillingQueueEventPage",
    );
  }

  /** The live state of queue rows, for a table that was read earlier. */
  public async getSummaries(ids: string[], options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueSummaries: unknown }>({
      document: BillingQueueSummariesDocument,
      operationName: "BillingQueueSummaries",
      variables: { ids },
      signal: options.signal,
    });
    return safeParse(
      z.array(billingQueueSummarySchema),
      data.billingQueueSummaries ?? [],
      "BillingQueueSummaries",
    );
  }

  public async resolveIssue(id: string, issueId: string, optionKey: string) {
    const data = await requestGraphQL<{ resolveBillingQueueIssue: unknown }>({
      document: ResolveBillingQueueIssueDocument,
      operationName: "ResolveBillingQueueIssue",
      variables: { id, issueId, optionKey },
    });
    return safeParse(billingQueueItemSchema, data.resolveBillingQueueIssue, "BillingQueueItem");
  }

  public async undoIssue(id: string, issueId: string) {
    const data = await requestGraphQL<{ undoBillingQueueIssue: unknown }>({
      document: UndoBillingQueueIssueDocument,
      operationName: "UndoBillingQueueIssue",
      variables: { id, issueId },
    });
    return safeParse(billingQueueItemSchema, data.undoBillingQueueIssue, "BillingQueueItem");
  }

  public async release(id: string) {
    const data = await requestGraphQL<{ releaseBillingQueueItem: unknown }>({
      document: ReleaseBillingQueueItemDocument,
      operationName: "ReleaseBillingQueueItem",
      variables: { id },
    });
    return safeParse(billingQueueItemSchema, data.releaseBillingQueueItem, "BillingQueueItem");
  }

  /** Posts the approved item's invoice, which sends it to the customer. */
  public async post(id: string) {
    const data = await requestGraphQL<{ postBillingQueueItem: unknown }>({
      document: PostBillingQueueItemDocument,
      operationName: "PostBillingQueueItem",
      variables: { id },
    });
    return safeParse(
      billingQueuePostResultSchema,
      data.postBillingQueueItem,
      "BillingQueuePostResult",
    );
  }

  public async redetectPODSignatures() {
    const data = await requestGraphQL<{ redetectBillingQueuePodSignatures: unknown }>({
      document: RedetectBillingQueuePodSignaturesDocument,
      operationName: "RedetectBillingQueuePodSignatures",
    });
    return safeParse(
      podSignatureRedetectResultSchema,
      data.redetectBillingQueuePodSignatures,
      "PODSignatureRedetectResult",
    );
  }

  /**
   * Starts one action over several items as a background run. Every action but posting
   * waits out a short undo window before it writes.
   */
  public async startBulk(
    action: BulkRunAction,
    itemIds: string[],
    idempotencyKey: string,
    params: BulkRunParams = {},
  ) {
    const data = await requestGraphQL<{ startBillingQueueBulkRun: unknown }>({
      document: StartBillingQueueBulkRunDocument,
      operationName: "StartBillingQueueBulkRun",
      variables: {
        input: {
          action: action as BillingQueueBulkAction,
          itemIds,
          idempotencyKey,
          params: {
            ...params,
            holdReasonCode: params.holdReasonCode as BillingQueueHoldReasonCode | undefined,
          },
        },
      },
    });
    return safeParse(
      billingQueueApprovalRunSchema,
      data.startBillingQueueBulkRun,
      "BillingQueueBulkRun",
    );
  }

  public async getBulk(runId: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueBulkRun: unknown }>({
      document: BillingQueueBulkRunDocument,
      operationName: "BillingQueueBulkRun",
      variables: { id: runId },
      signal: options.signal,
    });
    return safeParse(
      billingQueueApprovalRunSchema,
      data.billingQueueBulkRun,
      "BillingQueueBulkRun",
    );
  }

  public async undoBulk(runId: string) {
    const data = await requestGraphQL<{ cancelBillingQueueBulkRun: unknown }>({
      document: CancelBillingQueueBulkRunDocument,
      operationName: "CancelBillingQueueBulkRun",
      variables: { id: runId },
    });
    return safeParse(
      billingQueueApprovalRunSchema,
      data.cancelBillingQueueBulkRun,
      "BillingQueueBulkRun",
    );
  }

  public async listFilterPresets(options: RequestOptions = {}) {
    const data = await requestGraphQL<{ billingQueueFilterPresets: unknown }>({
      document: BillingQueueFilterPresetsDocument,
      operationName: "BillingQueueFilterPresets",
      signal: options.signal,
    });
    return safeParse(
      z.array(billingQueueFilterPresetSchema),
      data.billingQueueFilterPresets ?? [],
      "BillingQueueFilterPresets",
    );
  }

  public async createFilterPreset(payload: BillingQueueFilterPresetInput) {
    const data = await requestGraphQL<{ createBillingQueueFilterPreset: unknown }>({
      document: CreateBillingQueueFilterPresetDocument,
      operationName: "CreateBillingQueueFilterPreset",
      variables: { input: payload },
    });
    return safeParse(
      billingQueueFilterPresetSchema,
      data.createBillingQueueFilterPreset,
      "BillingQueueFilterPreset",
    );
  }

  public async updateFilterPreset(id: string, payload: BillingQueueFilterPresetInput) {
    const data = await requestGraphQL<{ updateBillingQueueFilterPreset: unknown }>({
      document: UpdateBillingQueueFilterPresetDocument,
      operationName: "UpdateBillingQueueFilterPreset",
      variables: { id, input: payload },
    });
    return safeParse(
      billingQueueFilterPresetSchema,
      data.updateBillingQueueFilterPreset,
      "BillingQueueFilterPreset",
    );
  }

  public async deleteFilterPreset(id: string) {
    await requestGraphQL<{ deleteBillingQueueFilterPreset: boolean }>({
      document: DeleteBillingQueueFilterPresetDocument,
      operationName: "DeleteBillingQueueFilterPreset",
      variables: { id },
    });
  }
}
