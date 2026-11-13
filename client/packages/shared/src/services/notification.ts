import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DataTablePageInfoFieldsFragmentDoc,
  DismissAllNotificationsDocument,
  DismissMyNotificationsDocument,
  DismissNotificationsDocument,
  MarkAllMyNotificationsReadDocument,
  MarkMyNotificationsReadDocument,
  MarkMyNotificationsUnreadDocument,
  MarkNotificationsReadDocument,
  MarkNotificationsUnreadDocument,
  MyNotificationListDocument,
  MyNotificationUnreadCountDocument,
  NotificationCountsDocument,
  NotificationListDocument,
  RestoreMyNotificationsDocument,
  RestoreNotificationsDocument,
  SnoozeNotificationsDocument,
  WakeNotificationsDocument,
  type MyNotificationListQuery,
  type MyNotificationUnreadCountQuery,
  type NotificationCountsQuery,
  type NotificationListQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import type { GraphQLExecutableDocument } from "@trenova/shared/types/graphql";
import {
  notificationCountsSchema,
  notificationFeedSchema,
  type Notification,
  type NotificationLens,
} from "@trenova/shared/types/notification";

/**
 * Notification visibility scope.
 * - "all": the back-office web app's panel, which also takes what went to a team.
 * - "mine": strictly the signed-in driver's own notifications (Dash portal);
 *   organization-wide notifications are never returned.
 */
export type NotificationScope = "all" | "mine";

export interface NotificationFeedParams {
  first?: number;
  after?: string | null;
  lens?: NotificationLens;
  unreadOnly?: boolean;
  scope?: NotificationScope;
}

type NotificationActionKind = "read" | "unread" | "dismiss" | "restore";

/**
 * Per-scope operation table for id-targeted notification actions. The "mine"
 * row hits driver-scoped mutations that resolve the worker server-side and can
 * only ever touch the driver's own notifications.
 */
const ACTION_OPERATIONS: Record<
  NotificationScope,
  Record<NotificationActionKind, { document: GraphQLExecutableDocument; operationName: string }>
> = {
  all: {
    read: { document: MarkNotificationsReadDocument, operationName: "MarkNotificationsRead" },
    unread: { document: MarkNotificationsUnreadDocument, operationName: "MarkNotificationsUnread" },
    dismiss: { document: DismissNotificationsDocument, operationName: "DismissNotifications" },
    restore: { document: RestoreNotificationsDocument, operationName: "RestoreNotifications" },
  },
  mine: {
    read: { document: MarkMyNotificationsReadDocument, operationName: "MarkMyNotificationsRead" },
    unread: {
      document: MarkMyNotificationsUnreadDocument,
      operationName: "MarkMyNotificationsUnread",
    },
    dismiss: { document: DismissMyNotificationsDocument, operationName: "DismissMyNotifications" },
    restore: { document: RestoreMyNotificationsDocument, operationName: "RestoreMyNotifications" },
  },
};

export class NotificationService {
  public async listNotifications(
    params?: NotificationFeedParams,
    options?: { signal?: AbortSignal },
  ) {
    const variables = {
      input: { first: params?.first ?? 30, after: params?.after ?? null },
      filter: {
        lens: params?.lens ?? "open",
        unreadOnly: params?.unreadOnly ?? false,
      },
    };

    const connection =
      params?.scope === "mine"
        ? (
            (await requestGraphQL({
              document: MyNotificationListDocument,
              operationName: "MyNotificationList",
              variables,
              signal: options?.signal,
            })) as MyNotificationListQuery
          ).myNotifications
        : (
            (await requestGraphQL({
              document: NotificationListDocument,
              operationName: "NotificationList",
              variables,
              signal: options?.signal,
            })) as NotificationListQuery
          ).notifications;

    const { edges, totalCount } = connection;
    const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

    return safeParse(
      notificationFeedSchema,
      {
        results: edges.map((edge) => edge.node),
        totalCount: totalCount ?? edges.length,
        endCursor: pageInfo.endCursor ?? null,
        hasNextPage: pageInfo.hasNextPage,
      },
      "Notification Feed",
    );
  }

  /** How many notifications wait in each part of the web app's panel. */
  public async getCounts(options?: { signal?: AbortSignal }) {
    const response = (await requestGraphQL({
      document: NotificationCountsDocument,
      operationName: "NotificationCounts",
      variables: {},
      signal: options?.signal,
    })) as NotificationCountsQuery;

    return safeParse(notificationCountsSchema, response.notificationCounts, "Notification Counts");
  }

  /** The driver portal's one number: the driver's own unread notifications. */
  public async getMyUnreadCount(options?: { signal?: AbortSignal }) {
    const response = (await requestGraphQL({
      document: MyNotificationUnreadCountDocument,
      operationName: "MyNotificationUnreadCount",
      variables: {},
      signal: options?.signal,
    })) as MyNotificationUnreadCountQuery;

    return response.myNotificationUnreadCount;
  }

  private async runAction(
    kind: NotificationActionKind,
    ids: Notification["id"][],
    scope: NotificationScope,
  ) {
    const { document, operationName } = ACTION_OPERATIONS[scope][kind];
    await requestGraphQL({ document, operationName, variables: { ids } });
  }

  public async markRead(ids: Notification["id"][], scope: NotificationScope = "all") {
    await this.runAction("read", ids, scope);
  }

  public async markUnread(ids: Notification["id"][], scope: NotificationScope = "all") {
    await this.runAction("unread", ids, scope);
  }

  /** The driver portal's "mark all read"; the web app marks a whole lens done instead. */
  public async markAllMineRead() {
    await requestGraphQL({
      document: MarkAllMyNotificationsReadDocument,
      operationName: "MarkAllMyNotificationsRead",
      variables: {},
    });
  }

  public async dismiss(ids: Notification["id"][], scope: NotificationScope = "all") {
    await this.runAction("dismiss", ids, scope);
  }

  /** Marks every open notification in one part of the panel done. */
  public async dismissAll(lens: NotificationLens) {
    await requestGraphQL({
      document: DismissAllNotificationsDocument,
      operationName: "DismissAllNotifications",
      variables: { lens },
    });
  }

  public async restore(ids: Notification["id"][], scope: NotificationScope = "all") {
    await this.runAction("restore", ids, scope);
  }

  /** Sets notifications aside until `until` (Unix seconds), when they come back unread. */
  public async snooze(ids: Notification["id"][], until: number) {
    await requestGraphQL({
      document: SnoozeNotificationsDocument,
      operationName: "SnoozeNotifications",
      variables: { ids, until },
    });
  }

  /** Brings set-aside notifications back now. */
  public async wake(ids: Notification["id"][]) {
    await requestGraphQL({
      document: WakeNotificationsDocument,
      operationName: "WakeNotifications",
      variables: { ids },
    });
  }
}

export const notificationService = new NotificationService();
