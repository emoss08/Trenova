import {
  AssignCollectionCollectorDocument,
  CollectionAccountDocument,
  CollectionWorklistDocument,
  LogCollectionActivityDocument,
  ScheduleCollectionFollowUpDocument,
  SendPaymentReminderDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  collectionAccountViewSchema,
  collectionWorklistPageSchema,
  type CollectionWorklistParams,
  type LogCollectionActivityInput,
  paymentReminderSendResultSchema,
} from "@/types/collection";

type RequestOptions = { signal?: AbortSignal };

function parseAccountView(view: unknown) {
  return safeParse(collectionAccountViewSchema, view, "CollectionAccountView");
}

export class CollectionService {
  /** The ranked worklist: the customers to chase first come first. */
  async worklist(params: CollectionWorklistParams, { signal }: RequestOptions = {}) {
    const response = await requestGraphQL({
      document: CollectionWorklistDocument,
      operationName: "CollectionWorklist",
      variables: { input: params },
      signal,
    });
    return safeParse(
      collectionWorklistPageSchema,
      response.collectionWorklist,
      "CollectionWorklistPage",
    );
  }

  /** One customer's account and a page of its history, older than the cursor. */
  async getAccount(
    customerId: string,
    before?: { createdAt: number; id: string },
    { signal }: RequestOptions = {},
  ) {
    const response = await requestGraphQL({
      document: CollectionAccountDocument,
      operationName: "CollectionAccount",
      variables: { customerId, before: before ?? null },
      signal,
    });
    return parseAccountView(response.collectionAccount);
  }

  async logActivity(customerId: string, input: LogCollectionActivityInput) {
    const response = await requestGraphQL({
      document: LogCollectionActivityDocument,
      operationName: "LogCollectionActivity",
      variables: { customerId, input },
    });
    return parseAccountView(response.logCollectionActivity);
  }

  async assignCollector(customerId: string, collectorId: string | null) {
    const response = await requestGraphQL({
      document: AssignCollectionCollectorDocument,
      operationName: "AssignCollectionCollector",
      variables: { customerId, collectorId },
    });
    return parseAccountView(response.assignCollectionCollector);
  }

  /** Emails the customer a reminder of everything overdue, one per currency. */
  async sendReminder(customerId: string, note: string) {
    const response = await requestGraphQL({
      document: SendPaymentReminderDocument,
      operationName: "SendPaymentReminder",
      variables: { customerId, note },
    });
    return safeParse(
      paymentReminderSendResultSchema,
      { reminders: response.sendPaymentReminder },
      "PaymentReminderSendResult",
    );
  }

  async scheduleFollowUp(customerId: string, followUpOn: number | null, note: string) {
    const response = await requestGraphQL({
      document: ScheduleCollectionFollowUpDocument,
      operationName: "ScheduleCollectionFollowUp",
      variables: { customerId, followUpOn, note },
    });
    return parseAccountView(response.scheduleCollectionFollowUp);
  }
}
