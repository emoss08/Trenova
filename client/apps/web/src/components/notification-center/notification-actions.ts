import {
  AcknowledgeWatchtowerItemDocument,
  ApproveWorkerPtoDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { NotificationRowAction } from "./notification-registry";

/** An action that changes a record, as against one that only opens a page. */
export type NotificationWriteAction = Exclude<NotificationRowAction, { kind: "open" }>;

export function isWriteAction(action: NotificationRowAction): action is NotificationWriteAction {
  return action.kind !== "open";
}

/**
 * Carries out a row's action through the same operation the record's own page uses, so
 * the permission check, the audit entry and the realtime update are the ones that page
 * already has. It resolves when the record has changed and rejects when it has not.
 */
export async function runNotificationAction(action: NotificationWriteAction): Promise<void> {
  switch (action.kind) {
    case "acknowledge-exception":
      await requestGraphQL({
        document: AcknowledgeWatchtowerItemDocument,
        operationName: "AcknowledgeWatchtowerItem",
        variables: { id: action.itemId },
      });
      return;
    case "approve-time-off":
      await requestGraphQL({
        document: ApproveWorkerPtoDocument,
        operationName: "ApproveWorkerPto",
        variables: { id: action.timeOffId },
      });
      return;
  }
}
