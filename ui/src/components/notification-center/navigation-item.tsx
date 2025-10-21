import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { faCheck, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { formatDistanceToNow } from "date-fns/formatDistanceToNow";
import React, { useState } from "react";
import { Icon } from "../ui/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";

interface NotificationItemProps {
  notification: any;
  onAction: (id: string, data?: any) => void;
  onMarkAsRead: () => void;
  onDismiss: () => void;
}

type NotificationItemOuterProps = React.ComponentProps<"div"> & {
  readAt?: boolean;
};

function NotificationItemOuter({
  children,
  readAt,
  ...props
}: NotificationItemOuterProps) {
  return (
    <div
      className={cn(
        "px-4 py-3 hover:bg-muted-foreground/10 transition-colors relative group",
        !readAt && "bg-muted/20",
      )}
      {...props}
    >
      <div className="flex gap-3">{children}</div>
    </div>
  );
}

export function NotificationItem({
  notification,
  // onAction,
  onMarkAsRead,
  onDismiss,
}: NotificationItemProps) {
  const [itemHover, setItemHover] = useState(false);

  return (
    <NotificationItemOuter
      readAt={notification.readAt}
      onMouseEnter={() => setItemHover(true)}
      onMouseLeave={() => setItemHover(false)}
      // onClick={() => onAction(notification.id, notification.data)}
    >
      <div className="flex-1 min-w-0 flex flex-col">
        <div className="flex items-center justify-between gap-2">
          <h4
            className={cn(
              "font-medium text-sm",
              notification.readAt && "text-muted-foreground",
            )}
          >
            {notification.title}
          </h4>
          {itemHover ? (
            <div className="flex h-6 items-center gap-1">
              <Tooltip delayDuration={200}>
                {!notification.readAt && (
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-6 hover:bg-muted-foreground/20 transition-colors"
                      onClick={(e) => {
                        e.stopPropagation();
                        onMarkAsRead();
                      }}
                    >
                      <Icon icon={faCheck} className="size-3" />
                    </Button>
                  </TooltipTrigger>
                )}
                <TooltipContent>Mark as read</TooltipContent>
              </Tooltip>
              <Tooltip delayDuration={200}>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-6 hover:bg-muted-foreground/20 transition-colors"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDismiss();
                    }}
                  >
                    <Icon icon={faXmark} className="size-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Dismiss</TooltipContent>
              </Tooltip>
            </div>
          ) : (
            <span className="flex h-6 items-center text-xs text-muted-foreground">
              {formatDistanceToNow(notification.createdAt * 1000, {
                addSuffix: true,
              })}
            </span>
          )}
        </div>
        <p className="text-sm mt-1 text-muted-foreground">
          {notification.message}
        </p>
      </div>
    </NotificationItemOuter>
  );
}
