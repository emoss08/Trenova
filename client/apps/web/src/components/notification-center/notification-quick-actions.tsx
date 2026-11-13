import {
  CheckIcon,
  ClockSnoozeIcon,
  DotsHorizontalIcon,
  FlipBackwardIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { DropdownMenuTrigger } from "@trenova/shared/components/ui/dropdown-menu";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Notification, NotificationLens } from "@trenova/shared/types/notification";
import type { MouseEvent, ReactElement, ReactNode } from "react";
import { laterTriggerId, type NotificationMenuHandle } from "./notification-row-menus";
import type { NotificationRowHandlers } from "./use-notification-row-handlers";

function stopPropagation(event: MouseEvent) {
  event.stopPropagation();
}

const QUICK_BUTTON = {
  type: "button",
  variant: "quiet",
  size: "icon-xs",
  className: "size-6 [&_svg]:size-3.5",
} as const;

function QuickAction({
  label,
  shortcut,
  trigger,
  children,
}: {
  label: string;
  shortcut?: string;
  /** The element the tooltip names: a plain button, or a menu's trigger. */
  trigger: ReactElement;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger render={trigger}>{children}</TooltipTrigger>
      <TooltipContent side="bottom" className="flex items-center gap-1.5">
        {label}
        {shortcut && <Kbd>{shortcut}</Kbd>}
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * The quick actions a row shows over its time on hover and on keyboard focus: done, later
 * and the rest. A finished row offers only the way back, and a row set aside has its own
 * button for that, so it shows none.
 */
export function NotificationQuickActions({
  notification,
  lens,
  handlers,
  laterMenu,
  moreMenu,
}: {
  notification: Notification;
  lens: NotificationLens;
  handlers: NotificationRowHandlers;
  laterMenu: NotificationMenuHandle;
  moreMenu: NotificationMenuHandle;
}) {
  const t = useT();
  if (lens === "later") return null;

  return (
    <div
      className="absolute top-1.5 right-1.5 flex gap-px rounded-md border border-border bg-overlay p-0.5 transition-opacity duration-100 pointer-fine:opacity-0 pointer-fine:group-focus-within/row:opacity-100 pointer-fine:group-hover/row:opacity-100 pointer-fine:has-[[data-popup-open]]:opacity-100"
      onClick={stopPropagation}
    >
      {lens === "done" ? (
        <QuickAction
          label={t("Move back")}
          trigger={
            <Button
              {...QUICK_BUTTON}
              aria-label={t("Move back")}
              onClick={() => handlers.restore(notification.id)}
            />
          }
        >
          <FlipBackwardIcon />
        </QuickAction>
      ) : (
        <>
          <QuickAction
            label={t("Done")}
            shortcut="E"
            trigger={
              <Button
                {...QUICK_BUTTON}
                aria-label={t("Done")}
                onClick={() => handlers.done(notification.id)}
              />
            }
          >
            <CheckIcon />
          </QuickAction>
          <QuickAction
            label={t("Later")}
            shortcut="H"
            trigger={
              <DropdownMenuTrigger
                id={laterTriggerId(notification.id)}
                handle={laterMenu}
                payload={notification}
                render={<Button {...QUICK_BUTTON} aria-label={t("Later")} />}
              />
            }
          >
            <ClockSnoozeIcon />
          </QuickAction>
          <QuickAction
            label={t("More")}
            trigger={
              <DropdownMenuTrigger
                handle={moreMenu}
                payload={notification}
                render={<Button {...QUICK_BUTTON} aria-label={t("More")} />}
              />
            }
          >
            <DotsHorizontalIcon />
          </QuickAction>
        </>
      )}
    </div>
  );
}
