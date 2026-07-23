import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  useMarkAllNotificationsRead,
  useNotificationAction,
  useNotificationFeed,
} from "@trenova/shared/hooks/use-notifications";
import { cn } from "@trenova/shared/lib/utils";
import type { Notification } from "@trenova/shared/types/notification";
import { BellIcon, CheckCheckIcon } from "lucide-react";
import { m } from "motion/react";
import { useNavigate } from "react-router";

function notificationLink(notification: Notification): string {
  const link = notification.data?.link;
  return typeof link === "string" && link.startsWith("/dash") ? link : "/dash";
}

function notificationTime(unix: number): string {
  const date = new Date(unix * 1000);
  const sameDay = new Date().toDateString() === date.toDateString();
  if (sameDay) {
    return new Intl.DateTimeFormat(undefined, {
      hour: "numeric",
      minute: "2-digit",
    }).format(date);
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(date);
}

const priorityDot: Record<string, string> = {
  critical: "bg-red-500",
  high: "bg-orange-500",
  medium: "bg-blue-500",
  low: "bg-muted-foreground/50",
};

export function DashNotificationsPage() {
  const navigate = useNavigate();
  const feed = useNotificationFeed({ state: "inbox", unreadOnly: false }, true);
  const markRead = useNotificationAction("read");
  const markAllRead = useMarkAllNotificationsRead();

  const notifications = feed.data?.notifications ?? [];
  const hasUnread = notifications.some((notification) => notification.readAt === null);

  const handleOpen = (notification: Notification) => {
    if (notification.readAt === null) {
      markRead.mutate([notification.id]);
    }
    void navigate(notificationLink(notification));
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight">Notifications</h1>
        {hasUnread ? (
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-xs text-muted-foreground"
            disabled={markAllRead.isPending}
            onClick={() => markAllRead.mutate()}
          >
            <CheckCheckIcon className="size-3.5" />
            Mark all read
          </Button>
        ) : null}
      </div>

      {feed.isPending ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-20 w-full rounded-2xl" />
          <Skeleton className="h-20 w-full rounded-2xl" />
          <Skeleton className="h-20 w-full rounded-2xl" />
        </div>
      ) : notifications.length > 0 ? (
        <>
          <ul className="flex flex-col gap-2">
            {notifications.map((notification, index) => {
              const unread = notification.readAt === null;
              return (
                <m.li
                  key={notification.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    duration: 0.2,
                    ease: "easeOut",
                    delay: Math.min(index * 0.03, 0.2),
                  }}
                >
                  <button
                    type="button"
                    onClick={() => handleOpen(notification)}
                    className={cn(
                      "flex w-full items-start gap-3 rounded-2xl border border-border bg-card p-4 text-left transition-colors hover:border-foreground/20",
                      unread && "border-primary/30 bg-primary/5",
                    )}
                  >
                    <span
                      className={cn(
                        "mt-1.5 size-2 shrink-0 rounded-full",
                        unread
                          ? (priorityDot[notification.priority] ?? "bg-blue-500")
                          : "bg-transparent",
                      )}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-baseline justify-between gap-2">
                        <span
                          className={cn(
                            "truncate text-sm",
                            unread ? "font-semibold" : "font-medium",
                          )}
                        >
                          {notification.title}
                        </span>
                        <span className="shrink-0 text-xs text-muted-foreground">
                          {notificationTime(notification.createdAt)}
                        </span>
                      </span>
                      <span className="mt-0.5 line-clamp-2 block text-xs text-muted-foreground">
                        {notification.message}
                      </span>
                    </span>
                  </button>
                </m.li>
              );
            })}
          </ul>
          {feed.hasNextPage ? (
            <Button
              variant="outline"
              size="sm"
              disabled={feed.isFetchingNextPage}
              onClick={() => void feed.fetchNextPage()}
            >
              {feed.isFetchingNextPage ? "Loading..." : "Load more"}
            </Button>
          ) : null}
        </>
      ) : (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-10 text-center">
          <BellIcon className="size-6 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            You&apos;re all caught up. Load assignments, settlements, and pay updates land here.
          </p>
        </div>
      )}
    </div>
  );
}
