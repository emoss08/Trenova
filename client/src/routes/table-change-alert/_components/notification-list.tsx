import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { formatTimestamp, getPriorityConfig, SOURCE_LABELS } from "@/lib/notification-helpers";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { Notification } from "@/types/notification";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckCheckIcon,
  CheckIcon,
  InboxIcon,
} from "lucide-react";
import { useCallback } from "react";
import { toast } from "sonner";

function NotificationRow({
  notification,
  onMarkRead,
}: {
  notification: Notification;
  onMarkRead: (id: string) => void;
}) {
  const isUnread = !notification.readAt;
  const config = getPriorityConfig(notification.priority);
  const sourceLabel = SOURCE_LABELS[notification.source] ?? notification.source;

  const data = notification.data as Record<string, string> | null;
  const operation = data?.operation;
  const tableName = data?.tableName;

  return (
    <div
      className={cn(
        "group flex items-start gap-3 rounded-lg border border-transparent px-4 py-3 transition-colors duration-150",
        isUnread ? "border-primary/10 bg-primary/[0.03]" : "hover:bg-accent/30",
      )}
    >
      <div className="mt-0.5 shrink-0">{config.icon}</div>

      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex items-start justify-between gap-3">
          <p
            className={cn(
              "text-sm leading-snug",
              isUnread ? "font-medium text-foreground" : "text-muted-foreground",
            )}
          >
            {notification.title}
          </p>
          {isUnread && (
            <span className={cn("mt-1.5 size-2 shrink-0 rounded-full", config.dot)} />
          )}
        </div>

        {notification.message && notification.message !== notification.title && (
          <p className="text-2xs leading-relaxed text-muted-foreground">{notification.message}</p>
        )}

        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant={config.badge} className="h-5 text-2xs">
            {sourceLabel}
          </Badge>
          {operation && (
            <Badge variant="outline" className="h-5 text-2xs">
              {operation}
            </Badge>
          )}
          {tableName && <span className="text-2xs text-muted-foreground">{tableName}</span>}
          <span className="text-2xs text-muted-foreground/60">
            {formatTimestamp(notification.createdAt)}
          </span>
        </div>
      </div>

      {isUnread && (
        <Button
          variant="ghost"
          size="icon-xs"
          className="mt-0.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
          onClick={() => onMarkRead(notification.id)}
          aria-label="Mark as read"
        >
          <CheckIcon className="size-3.5" />
        </Button>
      )}
    </div>
  );
}

export default function NotificationList() {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery(queries.notification.list({ limit: 50 }));
  const { data: unreadCount = 0 } = useQuery(queries.notification.unreadCount());

  const notifications = data?.results ?? [];

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

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-16">
        <InboxIcon className="size-6 animate-pulse text-muted-foreground/40" />
        <p className="text-sm text-muted-foreground">Loading notifications...</p>
      </div>
    );
  }

  if (notifications.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        <div className="flex size-14 items-center justify-center rounded-full bg-muted">
          <InboxIcon className="size-7 text-muted-foreground/60" />
        </div>
        <div className="text-center">
          <p className="text-sm font-medium text-foreground">No notifications yet</p>
          <p className="mt-1 max-w-sm text-2xs text-muted-foreground">
            When your subscriptions match database changes, notifications will appear here.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div>
      {unreadCount > 0 && (
        <>
          <div className="flex items-center justify-between py-2">
            <p className="text-2xs font-medium text-muted-foreground">
              {unreadCount} unread notification{unreadCount !== 1 ? "s" : ""}
            </p>
            <Button
              variant="ghost"
              size="xxs"
              className="text-2xs text-muted-foreground"
              onClick={handleMarkAllRead}
            >
              <CheckCheckIcon className="size-3" />
              Mark all read
            </Button>
          </div>
          <Separator className="mb-3" />
        </>
      )}

      <div className="flex flex-col gap-1">
        {notifications.map((n) => (
          <NotificationRow key={n.id} notification={n} onMarkRead={handleMarkRead} />
        ))}
      </div>
    </div>
  );
}
