import { useT } from "@trenova/shared/i18n/use-t";
import { CreateShipmentCommentDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { notification as notificationQueries } from "@trenova/shared/lib/queries/notification";
import { notificationService } from "@trenova/shared/services/notification";
import type { NotificationScope } from "@trenova/shared/services/notification";
import type {
  Notification,
  NotificationCounts,
  NotificationFeed,
  NotificationLens,
} from "@trenova/shared/types/notification";
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryClient,
} from "@tanstack/react-query";
import { toast } from "@trenova/shared/components/ui/toast";
import { defineLabels } from "@trenova/shared/i18n/labels";

const FEED_PAGE_SIZE = 30;
const COUNTS_REFETCH_INTERVAL = 60_000;

export interface NotificationFeedFilters {
  lens: NotificationLens;
  unreadOnly?: boolean;
}

type FeedData = InfiniteData<NotificationFeed, string | null>;

const FEEDS_KEY = [...notificationQueries.feed._def, "infinite"] as const;

function feedQueryKey(filters: NotificationFeedFilters, scope: NotificationScope) {
  return [...FEEDS_KEY, scope, filters] as const;
}

export function useNotificationFeed(
  filters: NotificationFeedFilters,
  enabled: boolean,
  scope: NotificationScope = "all",
) {
  return useInfiniteQuery({
    queryKey: feedQueryKey(filters, scope),
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam, signal }) =>
      notificationService.listNotifications(
        {
          first: FEED_PAGE_SIZE,
          after: pageParam,
          lens: filters.lens,
          unreadOnly: filters.unreadOnly ?? false,
          scope,
        },
        { signal },
      ),
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage && lastPage.endCursor ? lastPage.endCursor : undefined,
    enabled,
    select: (data) => ({
      notifications: data.pages.flatMap((page) => page.results),
      totalCount: data.pages[0]?.totalCount ?? 0,
    }),
  });
}

/** How many notifications wait in each part of the web app's panel. */
export function useNotificationCounts(enabled = true) {
  return useQuery({
    ...notificationQueries.counts(),
    refetchInterval: COUNTS_REFETCH_INTERVAL,
    enabled,
  });
}

/**
 * The number on the bell. In the web app it counts only what is the person's own and
 * unread: a broadcast to a team waits in the panel without lighting the bell, so the
 * bell means "something is yours". The driver portal has one list and counts all of it.
 */
export function useUnreadNotificationCount(scope: NotificationScope = "all") {
  const mine = useQuery({
    ...notificationQueries.myUnreadCount(),
    refetchInterval: COUNTS_REFETCH_INTERVAL,
    enabled: scope === "mine",
  });
  const counts = useNotificationCounts(scope === "all");

  return scope === "mine" ? mine : { ...counts, data: counts.data?.forYouUnread };
}

/** The part of the panel a notification is in at `now` (Unix seconds). */
export function lensOfNotification(
  notification: Pick<Notification, "dismissedAt" | "surfacesAt" | "reason">,
  now: number,
): Exclude<NotificationLens, "open"> {
  if (notification.dismissedAt !== null) return "done";
  if (notification.surfacesAt > now) return "later";
  return notification.reason === "team" ? "team" : "forYou";
}

function belongsInFeed(
  notification: Notification,
  filters: NotificationFeedFilters,
  now: number,
): boolean {
  const lens = lensOfNotification(notification, now);
  const inLens =
    filters.lens === "open" ? lens === "forYou" || lens === "team" : lens === filters.lens;

  return inLens && (!filters.unreadOnly || notification.readAt === null);
}

export type NotificationAction = "read" | "unread" | "dismiss" | "restore" | "wake";

/** What an action does to a notification, as the server will. */
export type NotificationPatch =
  | { action: NotificationAction }
  | { action: "snooze"; until: number };

export function patchNotification(
  notification: Notification,
  patch: NotificationPatch,
  now: number,
): Notification {
  switch (patch.action) {
    case "read":
      return { ...notification, readAt: notification.readAt ?? now };
    case "unread":
      return { ...notification, readAt: null };
    case "dismiss":
      return { ...notification, dismissedAt: now, readAt: notification.readAt ?? now };
    case "restore":
      return {
        ...notification,
        dismissedAt: null,
        resolvedAt: null,
        resolution: "",
        surfacesAt: now,
      };
    case "wake":
      return { ...notification, surfacesAt: now };
    case "snooze":
      return { ...notification, surfacesAt: patch.until, readAt: null };
  }
}

type CountDelta = Record<keyof NotificationCounts, number>;

function countsOf(notification: Notification, now: number): CountDelta {
  const lens = lensOfNotification(notification, now);
  const unread = notification.readAt === null ? 1 : 0;

  return {
    forYou: lens === "forYou" ? 1 : 0,
    forYouUnread: lens === "forYou" ? unread : 0,
    team: lens === "team" ? 1 : 0,
    teamUnread: lens === "team" ? unread : 0,
    later: lens === "later" ? 1 : 0,
  };
}

/**
 * Applies an action to every cached feed at once, drops each notification from the feeds
 * it no longer belongs in, and moves the tab counts by the same amount, so a row leaves
 * the list and its tab's number changes in the same frame. The server's answer replaces
 * all of it when the request settles.
 */
function applyOptimistic(queryClient: QueryClient, ids: Set<string>, patch: NotificationPatch) {
  const now = Math.floor(Date.now() / 1000);
  const delta: CountDelta = { forYou: 0, forYouUnread: 0, team: 0, teamUnread: 0, later: 0 };
  const counted = new Set<string>();
  let myUnreadDelta = 0;

  for (const query of queryClient.getQueryCache().findAll({ queryKey: FEEDS_KEY })) {
    const filters = query.queryKey.at(-1) as NotificationFeedFilters | undefined;
    if (!filters) continue;

    queryClient.setQueryData<FeedData>(query.queryKey, (data) => {
      if (!data) return data;

      return {
        ...data,
        pages: data.pages.map((page) => ({
          ...page,
          results: page.results.flatMap((item) => {
            if (!ids.has(item.id)) return [item];

            const next = patchNotification(item, patch, now);
            if (!counted.has(item.id)) {
              counted.add(item.id);
              const before = countsOf(item, now);
              const after = countsOf(next, now);
              for (const key of Object.keys(delta) as (keyof CountDelta)[]) {
                delta[key] += after[key] - before[key];
              }
              const wasUnread = item.dismissedAt === null && item.readAt === null;
              const isUnread = next.dismissedAt === null && next.readAt === null;
              myUnreadDelta += Number(isUnread) - Number(wasUnread);
            }

            return belongsInFeed(next, filters, now) ? [next] : [];
          }),
        })),
      };
    });
  }

  queryClient.setQueryData<NotificationCounts>(notificationQueries.counts().queryKey, (counts) =>
    counts
      ? {
          forYou: Math.max(0, counts.forYou + delta.forYou),
          forYouUnread: Math.max(0, counts.forYouUnread + delta.forYouUnread),
          team: Math.max(0, counts.team + delta.team),
          teamUnread: Math.max(0, counts.teamUnread + delta.teamUnread),
          later: Math.max(0, counts.later + delta.later),
        }
      : counts,
  );
  queryClient.setQueryData<number>(notificationQueries.myUnreadCount().queryKey, (count) =>
    count === undefined ? count : Math.max(0, count + myUnreadDelta),
  );
}

const ACTION_ERRORS: Record<NotificationAction | "snooze", string> = defineLabels({
  read: "Couldn't mark as read",
  unread: "Couldn't mark as unread",
  dismiss: "Couldn't mark the notification done",
  restore: "Couldn't move the notification back",
  wake: "Couldn't bring the notification back",
  snooze: "Couldn't set the notification aside",
});

const ACTION_FNS: Record<
  NotificationAction,
  (ids: string[], scope: NotificationScope) => Promise<void>
> = {
  read: (ids, scope) => notificationService.markRead(ids, scope),
  unread: (ids, scope) => notificationService.markUnread(ids, scope),
  dismiss: (ids, scope) => notificationService.dismiss(ids, scope),
  restore: (ids, scope) => notificationService.restore(ids, scope),
  wake: (ids) => notificationService.wake(ids),
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
      applyOptimistic(queryClient, new Set(ids), { action });
    },
    onError: () => {
      toast.error(ACTION_ERRORS[action]);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: notificationQueries._def }),
  });
}

/** Sets notifications aside until a time (Unix seconds); they come back unread. */
export function useSnoozeNotifications() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ ids, until }: { ids: string[]; until: number }) =>
      notificationService.snooze(ids, until),
    onMutate: async ({ ids, until }) => {
      await queryClient.cancelQueries({ queryKey: notificationQueries._def });
      applyOptimistic(queryClient, new Set(ids), { action: "snooze", until });
    },
    onError: () => {
      toast.error(ACTION_ERRORS.snooze);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: notificationQueries._def }),
  });
}

/** Marks every open notification in one part of the panel done. */
export function useDismissAllNotifications() {
  const t = useT();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (lens: NotificationLens) => notificationService.dismissAll(lens),
    onMutate: async (lens) => {
      await queryClient.cancelQueries({ queryKey: notificationQueries._def });
      const ids = new Set<string>();
      for (const [, data] of queryClient.getQueriesData<FeedData>({ queryKey: FEEDS_KEY })) {
        const now = Math.floor(Date.now() / 1000);
        for (const page of data?.pages ?? []) {
          for (const item of page.results) {
            if (lensOfNotification(item, now) === lens) ids.add(item.id);
          }
        }
      }
      applyOptimistic(queryClient, ids, { action: "dismiss" });
      queryClient.setQueryData<NotificationCounts>(
        notificationQueries.counts().queryKey,
        (counts) =>
          counts && (lens === "forYou" || lens === "team")
            ? { ...counts, [lens]: 0, [`${lens}Unread`]: 0 }
            : counts,
      );
    },
    onError: () => {
      toast.error(t("Couldn't mark them done"));
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
  const t = useT();

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
      toast.error(t("Couldn't send reply"));
    },
  });
}

/** The driver portal's "mark all read". The web app marks a whole lens done instead. */
export function useMarkAllMyNotificationsRead() {
  const t = useT();

  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => notificationService.markAllMineRead(),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: notificationQueries._def });
      const now = Math.floor(Date.now() / 1000);

      queryClient.setQueriesData<FeedData>({ queryKey: FEEDS_KEY }, (data) => {
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
      });
      queryClient.setQueryData<number>(notificationQueries.myUnreadCount().queryKey, 0);
    },
    onError: () => {
      toast.error(t("Couldn't mark all as read"));
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: notificationQueries._def }),
  });
}
