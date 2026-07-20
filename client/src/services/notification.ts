import { getFragmentData } from "@/graphql/generated";
import {
  DataTablePageInfoFieldsFragmentDoc,
  DismissNotificationsDocument,
  MarkAllNotificationsReadDocument,
  MarkNotificationsReadDocument,
  MarkNotificationsUnreadDocument,
  NotificationListDocument,
  NotificationUnreadCountDocument,
  RestoreNotificationsDocument,
  type NotificationListQuery,
  type NotificationUnreadCountQuery,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { safeParse } from "@/lib/parse";
import {
  notificationFeedSchema,
  type Notification,
  type NotificationState,
} from "@/types/notification";

export interface NotificationFeedParams {
  first?: number;
  after?: string | null;
  state?: NotificationState;
  unreadOnly?: boolean;
}

export class NotificationService {
  public async listNotifications(params?: NotificationFeedParams) {
    const response = (await requestGraphQL({
      document: NotificationListDocument,
      operationName: "NotificationList",
      variables: {
        input: { first: params?.first ?? 30, after: params?.after ?? null },
        filter: {
          state: params?.state ?? "inbox",
          unreadOnly: params?.unreadOnly ?? false,
        },
      },
    })) as NotificationListQuery;

    const { edges, totalCount } = response.notifications;
    const pageInfo = getFragmentData(
      DataTablePageInfoFieldsFragmentDoc,
      response.notifications.pageInfo,
    );

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

  public async markUnread(ids: Notification["id"][]) {
    await requestGraphQL({
      document: MarkNotificationsUnreadDocument,
      operationName: "MarkNotificationsUnread",
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

  public async dismiss(ids: Notification["id"][]) {
    await requestGraphQL({
      document: DismissNotificationsDocument,
      operationName: "DismissNotifications",
      variables: { ids },
    });
  }

  public async restore(ids: Notification["id"][]) {
    await requestGraphQL({
      document: RestoreNotificationsDocument,
      operationName: "RestoreNotifications",
      variables: { ids },
    });
  }
}
