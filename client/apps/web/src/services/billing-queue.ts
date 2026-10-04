import { z } from "zod";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  billingQueueApprovalRunSchema,
  billingQueueEventPageSchema,
  billingQueueFilterPresetSchema,
  billingQueueItemSchema,
  billingQueueNeighborsSchema,
  billingQueuePostResultSchema,
  billingQueueSummarySchema,
  billingQueueStatsSchema,
  reassignChargeResultSchema,
  type BillingQueueAssignInput,
  type BillingQueueFilterPreset,
  type BillingQueueApprovalRun,
  type BillingQueueEventPage,
  type BillingQueueFilterPresetInput,
  type BillingQueueItem,
  type BillingQueueNeighbors,
  type BillingQueuePostResult,
  type BillingQueueSummary,
  type BillingQueueStats,
  type BillingQueueUpdateChargesInput,
  type BillingQueueUpdateStatusInput,
  type ReassignChargeInput,
  type ReassignChargeResult,
} from "@trenova/shared/types/billing-queue";

export class BillingQueueService {
  public async getStats() {
    const response = await api.get<BillingQueueStats>("/billing-queue/stats/");
    return safeParse(billingQueueStatsSchema, response, "BillingQueueStats");
  }

  public async getById(id: string, params?: Record<string, string>) {
    const endpoint = params
      ? `/billing-queue/${id}/?${new URLSearchParams(params).toString()}`
      : `/billing-queue/${id}/`;
    const response = await api.get<BillingQueueItem>(endpoint);
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

  /** Starts a bulk approval; the server waits out the undo window before it writes. */
  public async startBulkApprove(itemIds: string[], idempotencyKey: string, assignMe = false) {
    const response = await api.post<BillingQueueApprovalRun>("/billing-queue/bulk-approve/", {
      itemIds,
      idempotencyKey,
      // Makes the person approving the biller of the items that have none.
      assignApprover: assignMe,
    });
    return safeParse(billingQueueApprovalRunSchema, response, "BillingQueueApprovalRun");
  }

  public async getBulkApprove(runId: string) {
    const response = await api.get<BillingQueueApprovalRun>(
      `/billing-queue/bulk-approve/${runId}/`,
    );
    return safeParse(billingQueueApprovalRunSchema, response, "BillingQueueApprovalRun");
  }

  public async undoBulkApprove(runId: string) {
    const response = await api.post<BillingQueueApprovalRun>(
      `/billing-queue/bulk-approve/${runId}/cancel/`,
      {},
    );
    return safeParse(billingQueueApprovalRunSchema, response, "BillingQueueApprovalRun");
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

  public async updateFilterPreset(id: string, payload: BillingQueueFilterPresetInput) {
    const response = await api.put<BillingQueueFilterPreset>(
      `/billing-queue/filter-presets/${id}/`,
      payload,
    );
    return safeParse(billingQueueFilterPresetSchema, response, "BillingQueueFilterPreset");
  }

  public async deleteFilterPreset(id: string) {
    await api.delete(`/billing-queue/filter-presets/${id}/`);
  }
}
