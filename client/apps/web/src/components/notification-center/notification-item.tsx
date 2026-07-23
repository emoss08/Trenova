import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatTimestamp } from "@/lib/notification-helpers";
import { cn } from "@/lib/utils";
import type { Notification } from "@/types/notification";
import { ArchiveIcon, ArchiveRestoreIcon, CheckIcon, MailIcon } from "lucide-react";
import type { KeyboardEvent } from "react";
import { NotificationContent } from "./notification-content";
import { getNotificationDescriptor, getNotificationLink } from "./notification-registry";

export interface NotificationItemActions {
  markRead: (ids: string[]) => void;
  markUnread: (ids: string[]) => void;
  archive: (ids: string[]) => void;
  restore: (ids: string[]) => void;
}

function ItemAction({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={label}
            onClick={(event) => {
              event.stopPropagation();
              onClick();
            }}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

export function NotificationItem({
  notification,
  actions,
  onNavigate,
}: {
  notification: Notification;
  actions: NotificationItemActions;
  onNavigate: (link: string) => void;
}) {
  const descriptor = getNotificationDescriptor(notification.eventType);
  const link = getNotificationLink(notification);
  const isUnread = notification.readAt === null;
  const isArchived = notification.dismissedAt !== null;
  const Icon = descriptor.icon;
  const avatar = descriptor.avatar?.(notification) ?? null;
  const canNavigate = Boolean(link) && !descriptor.disableRowNavigation;

  const activate = () => {
    if (isUnread) actions.markRead([notification.id]);
    if (canNavigate && link) onNavigate(link);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.target !== event.currentTarget) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      activate();
    }
  };

  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={isUnread ? `Unread: ${notification.title}` : notification.title}
      className={cn(
        "group relative flex w-full gap-3 px-4 py-3 text-left transition-colors duration-150 outline-none",
        "hover:bg-muted/40 focus-visible:bg-muted/40",
        canNavigate || isUnread ? "cursor-pointer" : "cursor-default",
      )}
      onClick={activate}
      onKeyDown={onKeyDown}
    >
      {avatar ? (
        <ResolvedUserAvatar
          userId={avatar.userId}
          name={avatar.name}
          className="mt-0.5 size-7 shrink-0"
          fallbackClassName="text-2xs"
          aria-hidden
        />
      ) : (
        <div
          className={cn(
            "mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md",
            descriptor.tileClass,
          )}
          aria-hidden
        >
          <Icon className={cn("size-3.5", descriptor.iconClass)} />
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-start justify-between gap-2">
          <p
            className={cn(
              "truncate text-xs leading-snug",
              isUnread ? "font-medium text-foreground" : "text-muted-foreground",
            )}
          >
            {notification.title}
          </p>
          <span className="flex shrink-0 items-center gap-1.5">
            <span className="text-2xs whitespace-nowrap text-muted-foreground/60 tabular-nums">
              {formatTimestamp(notification.createdAt)}
            </span>
            {isUnread && <span className="size-1.5 rounded-full bg-brand" aria-hidden />}
          </span>
        </div>

        {!descriptor.hideMessage &&
          notification.message &&
          notification.message !== notification.title && (
            <p className="mt-0.5 line-clamp-2 text-2xs leading-relaxed text-muted-foreground">
              {notification.message}
            </p>
          )}

        <NotificationContent notification={notification} onNavigate={onNavigate} />
      </div>

      <div
        className={cn(
          "absolute top-2 right-3 flex items-center gap-0.5 rounded-md border border-border bg-background p-0.5 shadow-sm",
          "opacity-0 transition-opacity duration-150 group-hover:opacity-100 focus-within:opacity-100",
        )}
      >
        {isArchived ? (
          <ItemAction label="Restore" onClick={() => actions.restore([notification.id])}>
            <ArchiveRestoreIcon className="size-3" />
          </ItemAction>
        ) : (
          <>
            {isUnread ? (
              <ItemAction label="Mark as read" onClick={() => actions.markRead([notification.id])}>
                <CheckIcon className="size-3" />
              </ItemAction>
            ) : (
              <ItemAction
                label="Mark as unread"
                onClick={() => actions.markUnread([notification.id])}
              >
                <MailIcon className="size-3" />
              </ItemAction>
            )}
            <ItemAction label="Archive" onClick={() => actions.archive([notification.id])}>
              <ArchiveIcon className="size-3" />
            </ItemAction>
          </>
        )}
      </div>
    </div>
  );
}
