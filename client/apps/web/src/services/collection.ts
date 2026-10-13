import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  collectionAccountViewSchema,
  collectionWorklistPageSchema,
  type CollectionAccountView,
  type CollectionWorklistPage,
  type CollectionWorklistParams,
  type LogCollectionActivityInput,
  type PaymentReminderSendResult,
  paymentReminderSendResultSchema,
} from "@/types/collection";

const BASE = "/accounting/collections";

export class CollectionService {
  /** The ranked worklist: the customers to chase first come first. */
  async worklist(params: CollectionWorklistParams) {
    const query = new URLSearchParams({
      scope: params.scope,
      dueOnly: String(params.dueOnly),
      search: params.search,
      limit: String(params.limit),
      offset: String(params.offset),
    });
    const response = await api.get<CollectionWorklistPage>(`${BASE}/worklist/?${query}`);
    return safeParse(collectionWorklistPageSchema, response, "CollectionWorklistPage");
  }

  /** One customer's account and a page of its history, older than the cursor. */
  async getAccount(customerId: string, before?: { createdAt: number; id: string }) {
    const query = before
      ? `?${new URLSearchParams({ beforeCreatedAt: String(before.createdAt), beforeId: before.id })}`
      : "";
    const response = await api.get<CollectionAccountView>(
      `${BASE}/customers/${customerId}/${query}`,
    );
    return safeParse(collectionAccountViewSchema, response, "CollectionAccountView");
  }

  async logActivity(customerId: string, input: LogCollectionActivityInput) {
    const response = await api.post<CollectionAccountView>(
      `${BASE}/customers/${customerId}/activities/`,
      input,
    );
    return safeParse(collectionAccountViewSchema, response, "CollectionAccountView");
  }

  async assignCollector(customerId: string, collectorId: string | null) {
    const response = await api.put<CollectionAccountView>(
      `${BASE}/customers/${customerId}/collector/`,
      { collectorId },
    );
    return safeParse(collectionAccountViewSchema, response, "CollectionAccountView");
  }

  /** Emails the customer a reminder of everything overdue, one per currency. */
  async sendReminder(customerId: string, note: string) {
    const response = await api.post<PaymentReminderSendResult>(
      `${BASE}/customers/${customerId}/reminder/`,
      { note },
    );
    return safeParse(paymentReminderSendResultSchema, response, "PaymentReminderSendResult");
  }

  async scheduleFollowUp(customerId: string, followUpOn: number | null, note: string) {
    const response = await api.put<CollectionAccountView>(
      `${BASE}/customers/${customerId}/follow-up/`,
      { followUpOn, note },
    );
    return safeParse(collectionAccountViewSchema, response, "CollectionAccountView");
  }
}
