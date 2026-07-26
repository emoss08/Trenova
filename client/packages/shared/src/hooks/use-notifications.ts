import { CreateShipmentCommentDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { notification as notificationQueries } from "@trenova/shared/lib/queries/notification";
import { notificationService } from "@trenova/shared/services/notification";
import type { NotificationScope } from "@trenova/shared/services/notification";
import type { Notification, NotificationFeed, NotificationState } from "@trenova/shared/types/notification";
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";

const FEED_PAGE_SIZE = 30;
const UNREAD_COUNT_REFETCH_INTERVAL = 60_000;

export interface NotificationFeedFilters {
  state: NotificationState;
  unreadOnly: boolean;
}

type FeedData = InfiniteData<NotificationFeed, string | null>;

function feedQueryKey(filters: NotificationFeedFilters, scope: NotificationScope) {
  return [...notificationQueries.feed._def, "infinite", scope, filters] as const;
}

export function useNotificationFeed(
  filters: NotificationFeedFilters,
  enabled: boolean,
  scope: NotificationScope = "all",
) {
  return useInfiniteQuery({
    queryKey: feedQueryKey(filters, scope),
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam }) =>
      notificationService.listNotifications({
        first: FEED_PAGE_SIZE,
        after: pageParam,
        state: filters.state,
        unreadOnly: filters.unreadOnly,
        scope,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage && lastPage.endCursor ? lastPage.endCursor : undefined,
    enabled,
    select: (data) => ({
      notifications: data.pages.flatMap((page) => page.results),
      totalCount: data.pages[0]?.totalCount ?? 0,
    }),
  });
}

export function useUnreadNotificationCount(scope: NotificationScope = "all") {
  return useQuery({
    ...notificationQueries.unreadCount(scope),
    refetchInterval: UNREAD_COUNT_REFETCH_INTERVAL,
  });
}

type NotificationAction = "read" | "unread" | "dismiss" | "restore";

function patchNotification(
  notification: Notification,
  action: NotificationAction,
  now: number,
): Notification {
  switch (action) {
    case "read":
      return { ...notification, readAt: notification.readAt ?? now };
    case "unread":
      return { ...notification, readAt: null };
    case "dismiss":
      return { ...notification, dismissedAt: now, readAt: notification.readAt ?? now };
    case "restore":
      return { ...notification, dismissedAt: null };
  }
}

function belongsInFeed(notification: Notification, filters: NotificationFeedFilters): boolean {
  if (filters.state === "archived") {
    return notification.dismissedAt !== null;
  }
  if (notification.dismissedAt !== null) {
    return false;
  }
  return !filters.unreadOnly || notification.readAt === null;
}

function applyFeedPatch(queryClient: QueryClient, ids: Set<string>, action: NotificationAction) {
  const now = Math.floor(Date.now() / 1000);
  let unreadDelta = 0;
  const counted = new Set<string>();

  queryClient.setQueriesData<FeedData>(
    { queryKey: [...notificationQueries.feed._def, "infinite"] },
    (data) => {
      if (!data) return data;

      return {
        ...data,
        pages: data.pages.map((page) => ({
          ...page,
          results: page.results.flatMap((item) => {
            if (!ids.has(item.id)) return [item];

            if (!counted.has(item.id)) {
              counted.add(item.id);
              if (action === "unread" && item.readAt !== null) unreadDelta += 1;
              if ((action === "read" || action === "dismiss") && item.readAt === null) {
                unreadDelta -= 1;
              }
            }

            return [patchNotification(item, action, now)];
          }),
        })),
      };
    },
  );

  return unreadDelta;
}

function pruneFeeds(queryClient: QueryClient) {
  const cache = queryClient.getQueryCache();
  for (const query of cache.findAll({
    queryKey: [...notificationQueries.feed._def, "infinite"],
  })) {
    const filters = query.queryKey.at(-1) as NotificationFeedFilters | undefined;
    if (!filters) continue;

    queryClient.setQueryData<FeedData>(query.queryKey, (data) => {
      if (!data) return data;
      return {
        ...data,
        pages: data.pages.map((page) => ({
          ...page,
          results: page.results.filter((item) => belongsInFeed(item, filters)),
        })),
      };
    });
  }
}

function adjustUnreadCount(queryClient: QueryClient, delta: number, scope: NotificationScope) {
  if (delta === 0) return;
  queryClient.setQueryData<number>(notificationQueries.unreadCount(scope).queryKey, (count) =>
    Math.max(0, (count ?? 0) + delta),
  );
}

const ACTION_ERRORS: Record<NotificationAction, string> = {
  read: "Couldn't mark as read",
  unread: "Couldn't mark as unread",
  dismiss: "Couldn't archive notification",
  restore: "Couldn't restore notification",
};

const ACTION_FNS: Record<
  NotificationAction,
  (ids: string[], scope: NotificationScope) => Promise<void>
> = {
  read: (ids, scope) => notificationService.markRead(ids, scope),
  unread: (ids, scope) => notificationService.markUnread(ids, scope),
  dismiss: (ids, scope) => notificationService.dismiss(ids, scope),
  restore: (ids, scope) => notificationService.restore(ids, scope),
};

export function useNotificationAction(
  action: NotificationAction,
  scope: NotificationScope = "all",
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ids: string[]) => ACTION_FNS[action](ids, scope),
    onMutate: async (ids) => {
      await queryClient.cancelQueries({ queryKey: notificationQueries._def });
      const delta = applyFeedPatch(queryClient, new Set(ids), action);
      adjustUnreadCount(queryClient, delta, scope);
      pruneFeeds(queryClient);
    },
    onError: () => {
      toast.error(ACTION_ERRORS[action]);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: notificationQueries._def }),
  });
}

export interface MentionReplyInput {
  shipmentId: string;
  comment: string;
  mentionUserId?: string | null;
}

export function useReplyToMention() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ shipmentId, comment, mentionUserId }: MentionReplyInput) =>
      requestGraphQL({
        document: CreateShipmentCommentDocument,
        operationName: "CreateShipmentComment",
        variables: {
          shipmentId,
          input: {
            comment,
            mentionedUserIds: mentionUserId ? [mentionUserId] : [],
          },
        },
      }),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["shipment-comments", variables.shipmentId],
      });
      void queryClient.invalidateQueries({
        queryKey: ["shipment-comment-count", variables.shipmentId],
      });
    },
    onError: () => {
      toast.error("Couldn't send reply");
    },
  });
}

export function useMarkAllNotificationsRead(scope: NotificationScope = "all") {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => notificationService.markAllRead(scope),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: notificationQueries._def });
      const now = Math.floor(Date.now() / 1000);

      queryClient.setQueriesData<FeedData>(
        { queryKey: [...notificationQueries.feed._def, "infinite"] },
        (data) => {
          if (!data) return data;
          return {
            ...data,
            pages: data.pages.map((page) => ({
              ...page,
              results: page.results.map((item) =>
                item.dismissedAt === null ? { ...item, readAt: item.readAt ?? now } : item,
              ),
            })),
          };
        },
      );
      queryClient.setQueryData<number>(notificationQueries.unreadCount(scope).queryKey, 0);
    },
    onError: () => {
      toast.error("Couldn't mark all as read");
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: notificationQueries._def }),
  });
}
