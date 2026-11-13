import { useUserTimezone } from "@/hooks/use-user-timezone";
import { snoozePresets, type SnoozePresetKey } from "@/lib/snooze-presets";
import { BellOff01Icon, CheckIcon, Link01Icon, Mail01Icon } from "@trenova/shared/components/icons";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  type DropdownMenuHandle,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useNow } from "@trenova/shared/hooks/use-now";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { Notification } from "@trenova/shared/types/notification";
import { useRef } from "react";
import { getNotificationLink, notificationPreferenceKey } from "./notification-registry";
import type { NotificationRowHandlers } from "./use-notification-row-handlers";

/** One menu for every row of the list: the row whose button was pressed is its payload. */
export type NotificationMenuHandle = DropdownMenuHandle<Notification>;

/** The id of a row's Later button, which the keyboard opens the menu from. */
export function laterTriggerId(notificationId: string): string {
  return `notification-later-${notificationId}`;
}

type LaterKey = "hour" | SnoozePresetKey;

const LATER_LABELS: Record<LaterKey, string> = defineLabels({
  hour: "In 1 hour",
  laterToday: "Later today",
  tomorrow: "Tomorrow morning",
  monday: "Monday",
});

const HOUR_SECONDS = 3600;
const WHEN: Intl.DateTimeFormatOptions = { weekday: "short", hour: "numeric", minute: "2-digit" };

/**
 * The times a row can be set aside until: an hour from now, then the working-day times
 * Triage and the Desk snooze to, in the person's own time zone.
 */
function laterOptions(now: number, timeZone: string): { key: LaterKey; until: number }[] {
  return [{ key: "hour", until: now + HOUR_SECONDS }, ...snoozePresets(now, timeZone)];
}

function LaterMenuContent({
  notification,
  onLater,
  nextRow,
}: {
  notification: Notification;
  onLater: NotificationRowHandlers["later"];
  nextRow: (notificationId: string) => HTMLElement | null;
}) {
  const t = useT();
  const now = useNow();
  const options = laterOptions(now, useUserTimezone());
  const next = useRef<HTMLElement | null>(null);

  // The row leaves once a time is picked, taking the button that opened the menu with it.
  // Someone working the list from the keyboard carries on from the row beside it, which is
  // found at the pick: by the time the menu has closed the row may already be gone.
  const finalFocus = (closeType: string) =>
    (closeType === "keyboard" && next.current?.isConnected && next.current) || true;

  return (
    <DropdownMenuContent align="end" className="w-56" finalFocus={finalFocus}>
      <DropdownMenuGroup>
        <DropdownMenuLabel>{t("Bring it back")}</DropdownMenuLabel>
        {options.map((option) => (
          <DropdownMenuItem
            key={option.key}
            title={t(LATER_LABELS[option.key])}
            endContent={
              <span className="text-xs text-muted-foreground tabular-nums">
                {formatUnixInUserTimezone(option.until, WHEN, "")}
              </span>
            }
            onClick={() => {
              next.current = nextRow(notification.id);
              onLater(notification.id, option.until);
            }}
          />
        ))}
      </DropdownMenuGroup>
    </DropdownMenuContent>
  );
}

export function NotificationLaterMenu({
  handle,
  onLater,
  nextRow,
}: {
  handle: NotificationMenuHandle;
  onLater: NotificationRowHandlers["later"];
  /** The row to carry on from once the given one has been set aside, if the list has one. */
  nextRow: (notificationId: string) => HTMLElement | null;
}) {
  return (
    <DropdownMenu<Notification> handle={handle}>
      {({ payload }) =>
        payload === undefined ? null : (
          <LaterMenuContent
            key={payload.id}
            notification={payload}
            onLater={onLater}
            nextRow={nextRow}
          />
        )
      }
    </DropdownMenu>
  );
}

function MoreMenuContent({
  notification,
  handlers,
}: {
  notification: Notification;
  handlers: NotificationRowHandlers;
}) {
  const t = useT();
  const link = getNotificationLink(notification);
  const canTurnOff = notificationPreferenceKey(notification) !== null;

  return (
    <DropdownMenuContent align="end" className="w-60">
      {notification.readAt === null ? (
        <DropdownMenuItem
          title={t("Mark as read")}
          startContent={<CheckIcon className="size-3.5" />}
          onClick={() => handlers.read(notification.id)}
        />
      ) : (
        <DropdownMenuItem
          title={t("Mark as unread")}
          startContent={<Mail01Icon className="size-3.5" />}
          onClick={() => handlers.unread(notification.id)}
        />
      )}
      {canTurnOff && (
        <DropdownMenuItem
          title={t("Turn off notifications like this")}
          startContent={<BellOff01Icon className="size-3.5" />}
          onClick={() => handlers.turnOff(notification)}
        />
      )}
      {link && (
        <DropdownMenuItem
          title={t("Copy link")}
          startContent={<Link01Icon className="size-3.5" />}
          onClick={() => handlers.copyLink(link)}
        />
      )}
    </DropdownMenuContent>
  );
}

export function NotificationMoreMenu({
  handle,
  handlers,
}: {
  handle: NotificationMenuHandle;
  handlers: NotificationRowHandlers;
}) {
  return (
    <DropdownMenu<Notification> handle={handle}>
      {({ payload }) =>
        payload === undefined ? null : (
          <MoreMenuContent notification={payload} handlers={handlers} />
        )
      }
    </DropdownMenu>
  );
}
