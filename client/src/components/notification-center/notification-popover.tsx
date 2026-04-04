import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { formatTimestamp, getPriorityConfig, SOURCE_LABELS } from "@/lib/notification-helpers";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { Notification } from "@/types/notification";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { BellIcon, CheckCheckIcon, CircleAlertIcon, InboxIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

function NotificationItem({
  notification,
  onMarkRead,
}: {
  notification: Notification;
  onMarkRead: (id: string) => void;
}) {
  const isUnread = !notification.readAt;
  const config = getPriorityConfig(notification.priority);
  const sourceLabel = SOURCE_LABELS[notification.source] ?? notification.source;

  return (
    <button
      type="button"
      aria-label={
        isUnread ? `Unread: ${notification.title}. Click to mark as read` : notification.title
      }
      className={cn(
        "group flex w-full gap-3 px-4 py-3 text-left transition-colors duration-150",
        "hover:bg-accent/50",
        isUnread && "bg-primary/[0.03]",
      )}
      onClick={() => {
        if (isUnread) onMarkRead(notification.id);
      }}
    >
      <div className="mt-0.5 shrink-0">{config.icon}</div>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <p
          className={cn(
            "truncate text-sm leading-snug",
            isUnread ? "font-medium text-foreground" : "text-muted-foreground",
          )}
        >
          {notification.title}
        </p>

        {notification.message && notification.message !== notification.title && (
          <p className="line-clamp-2 text-2xs leading-relaxed text-muted-foreground">
            {notification.message}
          </p>
        )}

        <div className="flex items-center gap-1.5">
          <Badge variant={config.badge} className="h-5 text-2xs">
            {sourceLabel}
          </Badge>
          <span className="text-2xs text-muted-foreground">
            {formatTimestamp(notification.createdAt)}
          </span>
        </div>
      </div>

      {isUnread && (
        <span
          className={cn("mt-2 size-2 shrink-0 rounded-full transition-opacity", config.dot)}
          aria-hidden
        />
      )}
    </button>
  );
}

export function NotificationPopover() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);

  const { data: unreadCount = 0 } = useQuery({
    ...queries.notification.unreadCount(),
    refetchInterval: 60_000,
  });

  const {
    data: notificationsData,
    isLoading,
    isError,
  } = useQuery({
    ...queries.notification.list({ limit: 20 }),
    enabled: open,
  });

  const notifications = notificationsData?.results ?? [];

  const invalidate = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: queries.notification._def });
  }, [queryClient]);

  const handleMarkRead = useCallback(
    async (id: string) => {
      try {
        await apiService.notificationService.markRead([id]);
        invalidate();
      } catch {
        toast.error("Failed to mark notification as read");
      }
    },
    [invalidate],
  );

  const handleMarkAllRead = useCallback(async () => {
    try {
      await apiService.notificationService.markAllRead();
      invalidate();
      toast.success("All notifications marked as read");
    } catch {
      toast.error("Failed to mark all as read");
    }
  }, [invalidate]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ""}`}
          >
            <span className="relative">
              <BellIcon className={cn("size-4 transition-colors", open && "text-foreground")} />
              {unreadCount > 0 && (
                <span className="absolute -top-1.5 -right-1.5 flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] leading-none font-semibold text-destructive-foreground ring-2 ring-background">
                  {unreadCount > 9 ? "9+" : unreadCount}
                </span>
              )}
            </span>
          </Button>
        }
      />
      <PopoverContent align="end" sideOffset={8} className="w-[420px] p-0">
        <div className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold">Notifications</h3>
            {unreadCount > 0 && (
              <Badge variant="info" className="h-5 text-2xs">
                {unreadCount} new
              </Badge>
            )}
          </div>
          {unreadCount > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="xxs"
              className="text-2xs text-muted-foreground"
              onClick={handleMarkAllRead}
            >
              <CheckCheckIcon className="size-3" />
              Mark all read
            </Button>
          )}
        </div>

        <Separator />

        <div className="max-h-[28rem] overflow-y-auto">
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <div className="flex flex-col items-center gap-2">
                <BellIcon className="size-5 animate-pulse text-muted-foreground/40" />
                <p className="text-2xs text-muted-foreground">Loading notifications...</p>
              </div>
            </div>
          )}

          {isError && (
            <div className="flex flex-col items-center justify-center gap-2 py-12">
              <CircleAlertIcon className="size-5 text-destructive/60" />
              <p className="text-2xs text-muted-foreground">Failed to load notifications</p>
            </div>
          )}

          {!isLoading && !isError && notifications.length === 0 && (
            <div className="flex flex-col items-center justify-center gap-3 py-12">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted">
                <InboxIcon className="size-6 text-muted-foreground/60" />
              </div>
              <div className="text-center">
                <p className="text-sm font-medium text-muted-foreground">All caught up</p>
                <p className="mt-0.5 text-2xs text-muted-foreground/70">
                  Notifications will appear here when your subscriptions match changes.
                </p>
              </div>
            </div>
          )}

          <div className="divide-y divide-border">
            {notifications.map((n) => (
              <NotificationItem key={n.id} notification={n} onMarkRead={handleMarkRead} />
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
