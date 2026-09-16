import { z } from "zod";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  billingQueueFilterPresetSchema,
  billingQueueItemSchema,
  billingQueueStatsSchema,
  reassignChargeResultSchema,
  type BillingQueueAssignInput,
  type BillingQueueFilterPreset,
  type BillingQueueFilterPresetInput,
  type BillingQueueItem,
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
