import { getFragmentData } from "@trenova/graphql/generated";
import {
  DataTablePageInfoFieldsFragmentDoc,
  DismissMyNotificationsDocument,
  DismissNotificationsDocument,
  MarkAllMyNotificationsReadDocument,
  MarkAllNotificationsReadDocument,
  MarkMyNotificationsReadDocument,
  MarkMyNotificationsUnreadDocument,
  MarkNotificationsReadDocument,
  MarkNotificationsUnreadDocument,
  MyNotificationListDocument,
  MyNotificationUnreadCountDocument,
  NotificationListDocument,
  NotificationUnreadCountDocument,
  RestoreMyNotificationsDocument,
  RestoreNotificationsDocument,
  type MyNotificationListQuery,
  type MyNotificationUnreadCountQuery,
  type NotificationListQuery,
  type NotificationUnreadCountQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import type { GraphQLExecutableDocument } from "@trenova/shared/types/graphql";
import {
  notificationFeedSchema,
  type Notification,
  type NotificationState,
} from "@trenova/shared/types/notification";

/**
 * Notification visibility scope.
 * - "all": organization feed (back-office web app).
 * - "mine": strictly the signed-in driver's own notifications (Dash portal);
 *   organization-wide notifications are never returned.
 */
export type NotificationScope = "all" | "mine";

export interface NotificationFeedParams {
  first?: number;
  after?: string | null;
  state?: NotificationState;
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
  Record<
    NotificationActionKind,
    { document: GraphQLExecutableDocument; operationName: string }
  >
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
  public async listNotifications(params?: NotificationFeedParams) {
    const variables = {
      input: { first: params?.first ?? 30, after: params?.after ?? null },
      filter: {
        state: params?.state ?? "inbox",
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
            })) as MyNotificationListQuery
          ).myNotifications
        : (
            (await requestGraphQL({
              document: NotificationListDocument,
              operationName: "NotificationList",
              variables,
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

  public async getUnreadCount(scope: NotificationScope = "all") {
    if (scope === "mine") {
      const response = (await requestGraphQL({
        document: MyNotificationUnreadCountDocument,
        operationName: "MyNotificationUnreadCount",
        variables: {},
      })) as MyNotificationUnreadCountQuery;

      return response.myNotificationUnreadCount;
    }

    const response = (await requestGraphQL({
      document: NotificationUnreadCountDocument,
      operationName: "NotificationUnreadCount",
      variables: {},
    })) as NotificationUnreadCountQuery;

    return response.notificationUnreadCount;
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

  public async markAllRead(scope: NotificationScope = "all") {
    if (scope === "mine") {
      await requestGraphQL({
        document: MarkAllMyNotificationsReadDocument,
        operationName: "MarkAllMyNotificationsRead",
        variables: {},
      });
      return;
    }

    await requestGraphQL({
      document: MarkAllNotificationsReadDocument,
      operationName: "MarkAllNotificationsRead",
      variables: {},
    });
  }

  public async dismiss(ids: Notification["id"][], scope: NotificationScope = "all") {
    await this.runAction("dismiss", ids, scope);
  }

  public async restore(ids: Notification["id"][], scope: NotificationScope = "all") {
    await this.runAction("restore", ids, scope);
  }
}

export const notificationService = new NotificationService();
