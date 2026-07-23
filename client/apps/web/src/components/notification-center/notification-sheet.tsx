import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsList, TabsTab } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useMarkAllNotificationsRead,
  useNotificationAction,
  useNotificationFeed,
  useUnreadNotificationCount,
} from "@/hooks/use-notifications";
import {
  getNotificationDayGroup,
  NOTIFICATION_DAY_GROUPS,
  type NotificationDayGroup,
} from "@/lib/notification-helpers";
import { cn } from "@/lib/utils";
import type { Notification, NotificationState } from "@/types/notification";
import { AnimatePresence, motion } from "motion/react";
import {
  ArchiveIcon,
  BellIcon,
  CheckCheckIcon,
  CircleAlertIcon,
  InboxIcon,
  MailCheckIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { NotificationItem, type NotificationItemActions } from "./notification-item";

function BellTrigger({ unreadCount, open }: { unreadCount: number; open: boolean }) {
  return (
    <span className="relative">
      <BellIcon className={cn("size-3 transition-colors", open && "text-foreground")} />
      {unreadCount > 0 && (
        <span className="absolute -top-2 -right-2 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-brand px-0.5 text-[9px] leading-none font-semibold text-brand-foreground tabular-nums">
          {unreadCount > 9 ? "9+" : unreadCount}
        </span>
      )}
    </span>
  );
}

function FeedSkeleton() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: 6 }, (_, index) => (
        <div key={index} className="flex gap-3 px-4 py-3">
          <Skeleton className="size-7 shrink-0 rounded-md" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3 w-3/5 rounded" />
            <Skeleton className="h-2.5 w-4/5 rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}

function FeedEmptyState({ state, unreadOnly }: { state: NotificationState; unreadOnly: boolean }) {
  const Icon = state === "archived" ? ArchiveIcon : unreadOnly ? MailCheckIcon : InboxIcon;
  const title =
    state === "archived"
      ? "Nothing archived"
      : unreadOnly
        ? "No unread notifications"
        : "You're all caught up";
  const description =
    state === "archived"
      ? "Notifications you archive are kept here."
      : unreadOnly
        ? "Everything in your inbox has been read."
        : "New activity that needs your attention will appear here.";

  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <div className="flex size-11 items-center justify-center rounded-full border border-border bg-muted/60">
        <Icon className="size-5 text-muted-foreground/70" />
      </div>
      <div className="text-center">
        <p className="text-xs font-medium text-foreground">{title}</p>
        <p className="mt-0.5 max-w-56 text-2xs text-muted-foreground/70">{description}</p>
      </div>
    </div>
  );
}

const EMPTY_FEED: Notification[] = [];

function groupNotifications(notifications: Notification[]) {
  const groups = new Map<NotificationDayGroup, Notification[]>();
  for (const item of notifications) {
    const group = getNotificationDayGroup(item.createdAt);
    const bucket = groups.get(group);
    if (bucket) {
      bucket.push(item);
    } else {
      groups.set(group, [item]);
    }
  }
  return NOTIFICATION_DAY_GROUPS.filter((group) => groups.has(group)).map((group) => ({
    label: group,
    items: groups.get(group)!,
  }));
}

export function NotificationSheet() {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<NotificationState>("inbox");
  const [unreadOnly, setUnreadOnly] = useState(false);

  const { data: unreadCount = 0 } = useUnreadNotificationCount();

  const filters = useMemo(
    () => ({ state: tab, unreadOnly: tab === "inbox" && unreadOnly }),
    [tab, unreadOnly],
  );

  const {
    data: feed,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useNotificationFeed(filters, open);

  const markRead = useNotificationAction("read");
  const markUnread = useNotificationAction("unread");
  const archive = useNotificationAction("dismiss");
  const restore = useNotificationAction("restore");
  const markAllRead = useMarkAllNotificationsRead();

  const actions = useMemo<NotificationItemActions>(
    () => ({
      markRead: markRead.mutate,
      markUnread: markUnread.mutate,
      archive: archive.mutate,
      restore: restore.mutate,
    }),
    [markRead.mutate, markUnread.mutate, archive.mutate, restore.mutate],
  );

  const handleNavigate = useCallback(
    (link: string) => {
      setOpen(false);
      void navigate(link);
    },
    [navigate],
  );

  const notifications = feed?.notifications ?? EMPTY_FEED;
  const groups = useMemo(() => groupNotifications(notifications), [notifications]);

  const sentinelRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasNextPage) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting) && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { rootMargin: "120px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage, groups.length]);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="xs"
            aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ""}`}
          />
        }
      >
        <BellTrigger unreadCount={unreadCount} open={open} />
      </SheetTrigger>

      <SheetContent
        side="right"
        className="w-[min(26rem,calc(100vw-2rem))] gap-0 overflow-hidden sm:max-w-none"
      >
        <div className="flex items-center justify-between gap-2 py-3 pr-11 pl-4">
          <div className="flex items-center gap-2">
            <SheetTitle className="text-sm font-semibold">Notifications</SheetTitle>
            {unreadCount > 0 && (
              <Badge variant="info" className="h-4.5 text-2xs tabular-nums">
                {unreadCount} new
              </Badge>
            )}
          </div>
          {unreadCount > 0 && tab === "inbox" && (
            <Button
              type="button"
              variant="ghost"
              size="xxs"
              className="text-2xs text-muted-foreground"
              onClick={() => markAllRead.mutate()}
            >
              <CheckCheckIcon className="size-3" />
              Mark all read
            </Button>
          )}
        </div>

        <div className="flex items-center justify-between border-b border-border pr-3 pl-2">
          <Tabs
            value={tab}
            onValueChange={(value) => setTab(value as NotificationState)}
            className="gap-0"
          >
            <TabsList variant="underline" className="py-0">
              <TabsTab value="inbox" className="h-8 px-2.5 text-xs sm:h-8 sm:text-xs">
                Inbox
              </TabsTab>
              <TabsTab value="archived" className="h-8 px-2.5 text-xs sm:h-8 sm:text-xs">
                Archive
              </TabsTab>
            </TabsList>
          </Tabs>
          {tab === "inbox" && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="xxs"
                    aria-pressed={unreadOnly}
                    className={cn(
                      "text-2xs text-muted-foreground",
                      unreadOnly && "bg-accent text-foreground",
                    )}
                    onClick={() => setUnreadOnly((value) => !value)}
                  />
                }
              >
                Unread
              </TooltipTrigger>
              <TooltipContent side="bottom">
                {unreadOnly ? "Show all notifications" : "Show unread only"}
              </TooltipContent>
            </Tooltip>
          )}
        </div>

        <ScrollArea className="min-h-0 flex-1" maskHeight={24}>
            {isLoading && <FeedSkeleton />}

            {isError && !isLoading && (
              <div className="flex flex-col items-center justify-center gap-3 py-16">
                <CircleAlertIcon className="size-5 text-destructive/60" />
                <p className="text-2xs text-muted-foreground">
                  Notifications couldn&apos;t be loaded.
                </p>
                <Button type="button" variant="outline" size="xs" onClick={() => void refetch()}>
                  Try again
                </Button>
              </div>
            )}

            {!isLoading && !isError && notifications.length === 0 && (
              <FeedEmptyState state={tab} unreadOnly={filters.unreadOnly} />
            )}

            {!isLoading &&
              groups.map((group) => (
                <div key={group.label}>
                  <p className="px-4 pt-3 pb-1 text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
                    {group.label}
                  </p>
                  <AnimatePresence initial={false}>
                    {group.items.map((notification) => (
                      <motion.div
                        key={notification.id}
                        layout="position"
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: "auto" }}
                        exit={{ opacity: 0, height: 0 }}
                        transition={{ duration: 0.18, ease: "easeOut" }}
                        className="overflow-hidden"
                      >
                        <NotificationItem
                          notification={notification}
                          actions={actions}
                          onNavigate={handleNavigate}
                        />
                      </motion.div>
                    ))}
                  </AnimatePresence>
                </div>
              ))}

            {hasNextPage && <div ref={sentinelRef} className="h-px w-full" />}
            {isFetchingNextPage && (
              <div className="flex items-center justify-center py-3">
                <Spinner className="size-3.5 text-muted-foreground" />
              </div>
            )}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
