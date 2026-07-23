import { CreateShipmentCommentDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { notification as notificationQueries } from "@/lib/queries/notification";
import { apiService } from "@/services/api";
import type { Notification, NotificationFeed, NotificationState } from "@/types/notification";
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

function feedQueryKey(filters: NotificationFeedFilters) {
  return [...notificationQueries.feed._def, "infinite", filters] as const;
}

export function useNotificationFeed(filters: NotificationFeedFilters, enabled: boolean) {
  return useInfiniteQuery({
    queryKey: feedQueryKey(filters),
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam }) =>
      apiService.notificationService.listNotifications({
        first: FEED_PAGE_SIZE,
        after: pageParam,
        state: filters.state,
        unreadOnly: filters.unreadOnly,
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

export function useUnreadNotificationCount() {
  return useQuery({
    ...notificationQueries.unreadCount(),
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

function adjustUnreadCount(queryClient: QueryClient, delta: number) {
  if (delta === 0) return;
  queryClient.setQueryData<number>(notificationQueries.unreadCount().queryKey, (count) =>
    Math.max(0, (count ?? 0) + delta),
  );
}

const ACTION_ERRORS: Record<NotificationAction, string> = {
  read: "Couldn't mark as read",
  unread: "Couldn't mark as unread",
  dismiss: "Couldn't archive notification",
  restore: "Couldn't restore notification",
};

const ACTION_FNS: Record<NotificationAction, (ids: string[]) => Promise<void>> = {
  read: (ids) => apiService.notificationService.markRead(ids),
  unread: (ids) => apiService.notificationService.markUnread(ids),
  dismiss: (ids) => apiService.notificationService.dismiss(ids),
  restore: (ids) => apiService.notificationService.restore(ids),
};

export function useNotificationAction(action: NotificationAction) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ACTION_FNS[action],
    onMutate: async (ids) => {
      await queryClient.cancelQueries({ queryKey: notificationQueries._def });
      const delta = applyFeedPatch(queryClient, new Set(ids), action);
      adjustUnreadCount(queryClient, delta);
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

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => apiService.notificationService.markAllRead(),
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
      queryClient.setQueryData<number>(notificationQueries.unreadCount().queryKey, 0);
    },
    onError: () => {
      toast.error("Couldn't mark all as read");
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: notificationQueries._def }),
  });
}
