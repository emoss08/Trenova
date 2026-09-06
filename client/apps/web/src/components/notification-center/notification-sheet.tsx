import { Button } from "@trenova/shared/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@trenova/shared/components/ui/sheet";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { cn } from "@trenova/shared/lib/utils";
import { BellIcon } from "lucide-react";
import { lazy, Suspense, useCallback, useState } from "react";
import { NotificationPanelSkeleton } from "./notification-skeletons";

// The feed body pulls the notification registry, the item renderer and the
// mention reply composer. Only the bell and its unread count belong in the
// chunk that renders the app header on every page.
const NotificationPanel = lazy(() => import("./notification-panel"));

function BellTrigger({ unreadCount, open }: { unreadCount: number; open: boolean }) {
  return (
    <span className="relative">
      <BellIcon className={cn("size-3 transition-colors", open && "text-foreground")} />
      {unreadCount > 0 && (
        <span className="bg-brand text-brand-foreground absolute -top-2 -right-2 flex h-3.5 min-w-3.5 items-center justify-center rounded-full px-0.5 text-[9px] leading-none font-semibold tabular-nums">
          {unreadCount > 9 ? "9+" : unreadCount}
        </span>
      )}
    </span>
  );
}

export function NotificationSheet() {
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const { data: unreadCount = 0 } = useUnreadNotificationCount();

  const handleOpenChange = useCallback((next: boolean) => {
    if (next) setMounted(true);
    setOpen(next);
  }, []);

  const handleClose = useCallback(() => setOpen(false), []);

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="xs"
            aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ""}`}
            onPointerEnter={() => void import("./notification-panel")}
          />
        }
      >
        <BellTrigger unreadCount={unreadCount} open={open} />
      </SheetTrigger>

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
