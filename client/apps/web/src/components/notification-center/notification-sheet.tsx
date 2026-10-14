import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@trenova/shared/components/ui/sheet";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { cn } from "@trenova/shared/lib/utils";
import { Bell01Icon } from "@trenova/shared/components/icons";
import { useAppDialogOpen, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { lazy, Suspense, useCallback, useState } from "react";
import { NotificationPanelSkeleton } from "./notification-skeletons";

// The feed body pulls the notification registry, the item renderer and the
// mention reply composer. Only the bell and its unread count belong in the
// chunk that renders the app header on every page.
const NotificationPanel = lazy(() => import("./notification-panel"));

function BellTrigger({
  unreadCount,
  open,
  iconClassName,
}: {
  unreadCount: number;
  open: boolean;
  iconClassName?: string;
}) {
  return (
    <span className="relative">
      <Bell01Icon
        className={cn("size-3 transition-colors", open && "text-foreground", iconClassName)}
      />
      {unreadCount > 0 && (
        <span className="bg-brand text-brand-foreground text-3xs absolute -top-2 -right-2 flex h-3.5 min-w-3.5 items-center justify-center rounded-full px-0.5 leading-none font-semibold tabular-nums">
          {unreadCount > 9 ? "9+" : unreadCount}
        </span>
      )}
    </span>
  );
}

/**
 * `className` restyles the bell for a shell that draws it as a tile rather
 * than a header button, and `tooltipSide` names it on hover there.
 */
export function NotificationSheet({
  className,
  iconClassName,
  tooltipSide,
}: {
  className?: string;
  iconClassName?: string;
  tooltipSide?: "right" | "bottom";
} = {}) {
  const t = useT();
  const open = useAppDialogOpen("notifications");
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const [mounted, setMounted] = useState(false);
  const { data: unreadCount = 0 } = useUnreadNotificationCount();

  if (open && !mounted) {
    setMounted(true);
  }

  const handleOpenChange = useCallback(
    (next: boolean) => setDialogOpen("notifications", next),
    [setDialogOpen],
  );

  const handleClose = useCallback(() => setDialogOpen("notifications", false), [setDialogOpen]);

  const label = unreadCount > 0 ? t("Notifications ({0} unread)", unreadCount) : t("Notifications");
  const trigger = (
    <SheetTrigger
      render={
        <Button
          type="button"
          variant="ghost"
          size="xs"
          aria-label={label}
          className={cn("w-6", className)}
          onPointerEnter={() => void import("./notification-panel")}
        />
      }
    >
      <BellTrigger unreadCount={unreadCount} open={open} iconClassName={iconClassName} />
    </SheetTrigger>
  );

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      {tooltipSide ? (
        <Tooltip>
          <TooltipTrigger render={trigger} />
          <TooltipContent side={tooltipSide} sideOffset={10}>
            {label}
          </TooltipContent>
        </Tooltip>
      ) : (
        trigger
      )}

      <SheetContent
        side="right"
        className="w-[min(26rem,calc(100vw-2rem))] gap-0 overflow-hidden sm:max-w-none"
      >
        {mounted ? (
          <Suspense fallback={<NotificationPanelSkeleton />}>
            <NotificationPanel open={open} unreadCount={unreadCount} onClose={handleClose} />
          </Suspense>
        ) : (
          <NotificationPanelSkeleton />
        )}
      </SheetContent>
    </Sheet>
  );
}
