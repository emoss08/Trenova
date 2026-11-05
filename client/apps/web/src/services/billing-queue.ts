import { z } from "zod";
import { api } from "@trenova/shared/lib/api";
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
  billingQueuePacketChecksSchema,
  billingQueuePacketPreviewSchema,
  billingQueuePostResultSchema,
  billingQueueSummarySchema,
  billingQueueStatsSchema,
  podSignatureRedetectResultSchema,
  reassignChargeResultSchema,
  type BillingQueueAssignInput,
  type BillingQueueAssist,
  type BillingQueueBrief,
  type BillingQueueNovaGeneration,
  type BillingQueueInvoiceDetailsInput,
  type BillingQueueRecommendation,
  type BillingQueueSearchIntent,
  type BulkRunAction,
  type BulkRunParams,
  type BillingQueueFilterPreset,
  type BillingQueueApprovalRun,
  type BillingQueueEventPage,
  type BillingQueueFilterPresetInput,
  type BillingQueueItem,
  type BillingQueueNeighbors,
  type BillingQueuePacketChecks,
  type BillingQueuePacketPreview,
  type BillingQueuePostResult,
  type BillingQueueSummary,
  type BillingQueueStats,
  type BillingQueueUpdateChargesInput,
  type BillingQueueUpdateStatusInput,
  type PODSignatureRedetectResult,
  type ReassignChargeInput,
  type ReassignChargeResult,
} from "@trenova/shared/types/billing-queue";

/** The packet approving the item would send, as one PDF. */
export function billingQueuePacketEndpoint(itemId: string): string {
  return `/billing-queue/${encodeURIComponent(itemId)}/packet-preview/`;
}

export function billingQueuePacketPageEndpoint(itemId: string, page: number): string {
  return `${billingQueuePacketEndpoint(itemId)}pages/${page}/image/`;
}

export function billingQueuePacketThumbnailEndpoint(itemId: string, page: number): string {
  return `${billingQueuePacketEndpoint(itemId)}pages/${page}/thumbnail/`;
}

export class BillingQueueService {
  /** Lane counts, plus the pulse of the caller's day when `dayStart` is given. */
  public async getStats(dayStart?: number, options: { signal?: AbortSignal } = {}) {
    const endpoint = dayStart
      ? `/billing-queue/stats/?dayStart=${dayStart}`
      : "/billing-queue/stats/";
    const response = await api.get<BillingQueueStats>(endpoint, { signal: options.signal });
    return safeParse(billingQueueStatsSchema, response, "BillingQueueStats");
  }

  public async getRecommendations(options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueueRecommendation[]>(
      "/billing-queue/recommendations/",
      { signal: options.signal },
    );
    return safeParse(
      z.array(billingQueueRecommendationSchema),
      response ?? [],
      "BillingQueueRecommendations",
    );
  }

  public async getAssist(options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueueAssist>("/billing-queue/assist/", {
      signal: options.signal,
    });
    return safeParse(billingQueueAssistSchema, response, "BillingQueueAssist");
  }

  public async getBrief(options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueueBrief>("/billing-queue/brief/", {
      signal: options.signal,
    });
    return safeParse(billingQueueBriefSchema, response, "BillingQueueBrief");
  }

  /**
   * What Nova is generating for the whole queue. Reading it starts a generation
   * when the queue has items Nova has not reviewed or no brief.
   */
  public async getNovaGeneration(options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueueNovaGeneration>("/billing-queue/nova-generation/", {
      signal: options.signal,
    });
    return safeParse(billingQueueNovaGenerationSchema, response, "BillingQueueNovaGeneration");
  }

  /** Reads the filters out of a sentence typed into the queue's search; changes nothing. */
  /** Turns a sentence into queue filters; `today` (YYYY-MM-DD) is the person's own date. */
  public async composeSearch(text: string, today: string) {
    const response = await api.post<BillingQueueSearchIntent>("/billing-queue/search-intent/", {
      text,
      today,
    });
    return safeParse(billingQueueSearchIntentSchema, response, "BillingQueueSearchIntent");
  }

  public async updateInvoiceDetails(id: string, payload: BillingQueueInvoiceDetailsInput) {
    const response = await api.put<BillingQueueItem>(
      `/billing-queue/${id}/invoice-details/`,
      payload,
    );
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  public async getById(
    id: string,
    params?: Record<string, string>,
    options: { signal?: AbortSignal } = {},
  ) {
    const endpoint = params
      ? `/billing-queue/${id}/?${new URLSearchParams(params).toString()}`
      : `/billing-queue/${id}/`;
    const response = await api.get<BillingQueueItem>(endpoint, { signal: options.signal });
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  public async updateStatus(id: string, payload: BillingQueueUpdateStatusInput) {
    const response = await api.put<BillingQueueItem>(`/billing-queue/${id}/status/`, payload);
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  public async assign(id: string, payload: BillingQueueAssignInput) {
    const response = await api.put<BillingQueueItem>(`/billing-queue/${id}/assign/`, payload);
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  public async updateCharges(id: string, payload: BillingQueueUpdateChargesInput) {
    const response = await api.put<BillingQueueItem>(`/billing-queue/${id}/charges/`, payload);
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  /**
   * Changes who pays for one charge on the item's shipment while it is in
   * review. The server creates or cancels sibling queue items as payers gain or
   * lose a share.
   */
  public async reassignCharge(id: string, payload: ReassignChargeInput) {
    const response = await api.post<ReassignChargeResult>(
      `/billing-queue/${id}/reassign-charge/`,
      payload,
    );
    return safeParse(reassignChargeResultSchema, response, "ReassignChargeResult");
  }

  /** The item's place in the queue: what comes before and after it, and where it sits. */
  public async getNeighbors(id: string, params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    const response = await api.get<BillingQueueNeighbors>(
      `/billing-queue/${id}/neighbors/${query}`,
    );
    return safeParse(billingQueueNeighborsSchema, response, "BillingQueueNeighbors");
  }

  /** The item's activity, newest first, a page at a time. */
  public async getActivity(id: string, before?: { at: number; id: string }, limit = 30) {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) {
      params.set("beforeAt", String(before.at));
      params.set("beforeId", before.id);
    }
    const response = await api.get<BillingQueueEventPage>(
      `/billing-queue/${id}/activity/?${params.toString()}`,
    );
    return safeParse(billingQueueEventPageSchema, response, "BillingQueueEventPage");
  }

  /** The live state of queue rows, for a table that was read earlier. */
  public async getSummaries(ids: string[]) {
    const response = await api.get<{ results: BillingQueueSummary[] }>(
      `/billing-queue/summaries/?ids=${encodeURIComponent(ids.join(","))}`,
    );
    const parsed = await safeParse(
      z.object({ results: z.array(billingQueueSummarySchema) }),
      response,
      "BillingQueueSummaries",
    );
    return parsed.results;
  }

  public async resolveIssue(id: string, issueId: string, optionKey: string) {
    const response = await api.post<BillingQueueItem>(
      `/billing-queue/${id}/issues/${issueId}/resolve/`,
      { optionKey },
    );
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  public async undoIssue(id: string, issueId: string) {
    const response = await api.post<BillingQueueItem>(
      `/billing-queue/${id}/issues/${issueId}/undo/`,
      {},
    );
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  public async release(id: string) {
    const response = await api.post<BillingQueueItem>(`/billing-queue/${id}/release/`, {});
    return safeParse(billingQueueItemSchema, response, "BillingQueueItem");
  }

  /** Posts the approved item's invoice, which sends it to the customer. */
  public async post(id: string) {
    const response = await api.post<BillingQueuePostResult>(`/billing-queue/${id}/post/`, {});
    return safeParse(billingQueuePostResultSchema, response, "BillingQueuePostResult");
  }

  public async redetectPODSignatures() {
    const response = await api.post<PODSignatureRedetectResult>(
      "/billing-queue/pod-signatures/redetect/",
      {},
    );
    return safeParse(podSignatureRedetectResultSchema, response, "PODSignatureRedetectResult");
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
    const response = await api.post<BillingQueueApprovalRun>("/billing-queue/bulk/", {
      action,
      itemIds,
      idempotencyKey,
      params,
    });
    return safeParse(billingQueueApprovalRunSchema, response, "BillingQueueBulkRun");
  }

  /** Assembles the packet approving the item would send and says how many pages it has. */
  public async getPacketPreview(itemId: string, options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueuePacketPreview>(
      `${billingQueuePacketEndpoint(itemId)}summary/`,
      { signal: options.signal },
    );
    return safeParse(billingQueuePacketPreviewSchema, response, "BillingQueuePacketPreview");
  }

  /** Which document pages of the packet look blank, measured once per assembly. */
  public async getPacketChecks(itemId: string, options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueuePacketChecks>(
      `${billingQueuePacketEndpoint(itemId)}checks/`,
      { signal: options.signal },
    );
    return safeParse(billingQueuePacketChecksSchema, response, "BillingQueuePacketChecks");
  }

  public async getBulk(runId: string, options: { signal?: AbortSignal } = {}) {
    const response = await api.get<BillingQueueApprovalRun>(`/billing-queue/bulk/${runId}/`, {
      signal: options.signal,
    });
    return safeParse(billingQueueApprovalRunSchema, response, "BillingQueueBulkRun");
  }

  public async undoBulk(runId: string) {
    const response = await api.post<BillingQueueApprovalRun>(
      `/billing-queue/bulk/${runId}/cancel/`,
      {},
    );
    return safeParse(billingQueueApprovalRunSchema, response, "BillingQueueBulkRun");
  }

  public async listFilterPresets() {
    const response = await api.get<{ results: BillingQueueFilterPreset[]; count: number }>(
      "/billing-queue/filter-presets/",
    );
    const parsed = await safeParse(
      z.object({ results: z.array(billingQueueFilterPresetSchema), count: z.number() }),
      response,
      "BillingQueueFilterPresets",
    );
    return parsed.results;
  }

  public async createFilterPreset(payload: BillingQueueFilterPresetInput) {
    const response = await api.post<BillingQueueFilterPreset>(
      "/billing-queue/filter-presets/",
      payload,
    );
    return safeParse(billingQueueFilterPresetSchema, response, "BillingQueueFilterPreset");
  }

  public async deleteFilterPreset(id: string) {
    await api.delete(`/billing-queue/filter-presets/${id}/`);
  }
}

export const billingQueueService = new BillingQueueService();
