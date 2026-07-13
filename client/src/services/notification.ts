import {
  MarkAllNotificationsReadDocument,
  MarkNotificationsReadDocument,
  NotificationListDocument,
  NotificationUnreadCountDocument,
  type NotificationListQuery,
  type NotificationUnreadCountQuery,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { safeParse } from "@/lib/parse";
import { notificationResponseSchema, type Notification } from "@/types/notification";

export class NotificationService {
  public async listNotifications(params?: { limit?: number; offset?: number }) {
    const response = (await requestGraphQL({
      document: NotificationListDocument,
      operationName: "NotificationList",
      variables: { input: { first: params?.limit ?? 20 } },
    })) as NotificationListQuery;

    const results = response.notifications.edges.map((edge) => edge.node);
    const totalCount = response.notifications.totalCount;

    return safeParse(
      notificationResponseSchema,
      {
        results,
        count: totalCount ?? results.length,
        next: null,
        prev: null,
      },
      "Notification List",
    );
  }

  public async getUnreadCount() {
    const response = (await requestGraphQL({
      document: NotificationUnreadCountDocument,
      operationName: "NotificationUnreadCount",
      variables: {},
    })) as NotificationUnreadCountQuery;

    return response.notificationUnreadCount;
  }

  public async markRead(ids: Notification["id"][]) {
    await requestGraphQL({
      document: MarkNotificationsReadDocument,
      operationName: "MarkNotificationsRead",
      variables: { ids },
    });
  }

  public async markAllRead() {
    await requestGraphQL({
      document: MarkAllNotificationsReadDocument,
      operationName: "MarkAllNotificationsRead",
      variables: {},
    });
  }
}
