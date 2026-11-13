import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { CheckIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Notification, NotificationLens } from "@trenova/shared/types/notification";
import { useState, type KeyboardEvent, type MouseEvent } from "react";
import { isWriteAction } from "./notification-actions";
import { NotificationContent, NotificationReply } from "./notification-content";
import { NotificationQuickActions } from "./notification-quick-actions";
import {
  getNotificationLink,
  notificationActions,
  notificationRecent,
  notificationStatus,
  notificationWhy,
  resolveNotificationDescriptor,
  type NotificationOccurrence,
  type NotificationRowAction,
} from "./notification-registry";
import type { NotificationMenuHandle } from "./notification-row-menus";
import type { NotificationRowHandlers } from "./use-notification-row-handlers";

/** How a notification cleared itself, by the code the server stored when it did. */
const RESOLUTIONS = defineLabels({
  cleared: "The condition it reported no longer holds",
  resolved: "The exception was resolved",
  decided: "The changes were decided",
  other: "It cleared itself",
});

function resolutionLabel(code: string): string {
  return Object.hasOwn(RESOLUTIONS, code)
    ? RESOLUTIONS[code as keyof typeof RESOLUTIONS]
    : RESOLUTIONS.other;
}

const WEEK_SECONDS = 7 * 24 * 60 * 60;
const CLOCK: Intl.DateTimeFormatOptions = { hour: "numeric", minute: "2-digit" };

/**
 * When a row set aside comes back, as its badge says it: the day of the week and the time
 * in the person's own time zone, or the date once it is more than a week off.
 */
function comesBack(t: TranslateFn, surfacesAt: number, now: number): string {
  return t(
    "Back {0}",
    formatUnixInUserTimezone(
      surfacesAt,
      surfacesAt - now < WEEK_SECONDS
        ? { weekday: "long", ...CLOCK }
        : { month: "short", day: "numeric", ...CLOCK },
      "",
    ),
  );
}

const STATE_BADGE = "h-4.5 px-1.5 text-2xs";

function stopPropagation(event: MouseEvent) {
  event.stopPropagation();
}

function RowAction({
  notification,
  action,
  solid,
  handlers,
}: {
  notification: Notification;
  action: NotificationRowAction;
  solid: boolean;
  handlers: NotificationRowHandlers;
}) {
  const t = useT();
  const [busy, setBusy] = useState(false);

  const press = () => {
    if (!isWriteAction(action)) {
      handlers.open(notification, action.href, action.settles);
      return;
    }

    setBusy(true);
    void handlers.run(notification, action).finally(() => setBusy(false));
  };

  return (
    <Button
      type="button"
      size="xs"
      variant={solid ? "default" : "secondary"}
      className="text-xs"
      isLoading={busy}
      onClick={press}
    >
      {t(action.label)}
    </Button>
  );
}

function RecentOccurrences({
  notification,
  occurrences,
  now,
  handlers,
}: {
  notification: Notification;
  occurrences: NotificationOccurrence[];
  now: number;
  handlers: NotificationRowHandlers;
}) {
  return (
    <ul className="mt-2 divide-y divide-border-subtle border-t border-border-subtle">
      {occurrences.map((occurrence) => {
        const { link } = occurrence;

        return (
          <li
            key={`${occurrence.at}:${occurrence.label}`}
            className="flex items-baseline gap-2 py-1.5 text-xs"
          >
            {occurrence.ref && (
              <span className="shrink-0 font-mono text-muted-foreground">{occurrence.ref}</span>
            )}
            {link ? (
              <button
                type="button"
                className="min-w-0 flex-1 cursor-pointer truncate rounded-sm ui-focus-ring text-left text-foreground hover:underline"
                onClick={() => handlers.open(notification, link)}
              >
                {occurrence.label}
              </button>
            ) : (
              <span className="min-w-0 flex-1 truncate text-foreground">{occurrence.label}</span>
            )}
            <span className="shrink-0 text-foreground-subtle tabular-nums">
              {formatRelativeTime(occurrence.at - now, "narrow")}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

/**
 * One notification: what it is about, why the person got it, and what can be done about
 * it without leaving the panel. Opening it goes to the record; the quick actions over its
 * time finish it, set it aside or bring it back.
 */
export function NotificationItem({
  notification,
  lens,
  now,
  leaving,
  handlers,
  laterMenu,
  moreMenu,
}: {
  notification: Notification;
  /** The part of the panel the row is drawn in, which decides what it offers. */
  lens: NotificationLens;
  /** Unix seconds, shared by every row so they age together. */
  now: number;
  /** Whether the row is folding away before it leaves the list. */
  leaving: boolean;
  handlers: NotificationRowHandlers;
  laterMenu: NotificationMenuHandle;
  moreMenu: NotificationMenuHandle;
}) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);

  const descriptor = resolveNotificationDescriptor(notification);
  const link = getNotificationLink(notification);
  const done = lens === "done";
  const later = lens === "later";
  // A row set aside comes back unread; until then it is not asking to be read.
  const isUnread = notification.readAt === null && !done && !later;
  const Icon = done ? CheckIcon : descriptor.icon;
  const avatar = done ? null : (descriptor.avatar?.(notification) ?? null);
  const title = t(notification.title);

  const when = done
    ? (notification.dismissedAt ?? notification.surfacesAt)
    : notification.surfacesAt;
  const status = done ? null : notificationStatus(notification);
  const why =
    notificationWhy(notification) ?? (notification.reason === "team" ? descriptor.category : null);
  const resolution = notification.resolvedAt === null ? null : notification.resolution;
  const actions = done || later ? [] : notificationActions(notification);
  const recent = done ? [] : notificationRecent(notification);
  const showMessage =
    !descriptor.hideMessage && notification.message !== "" && notification.message !== title;
  const critical = !done && notification.priority === "critical";
  const note = done
    ? resolution === null
      ? t("You marked it done")
      : t(resolutionLabel(resolution))
    : why === null
      ? null
      : t(why);

  const activate = () => {
    if (link) handlers.open(notification, link);
    else if (isUnread) handlers.read(notification.id);
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
      className={cn(
        "grid transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none",
        leaving ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr]",
      )}
    >
      <div className={cn("min-h-0", leaving && "overflow-hidden")}>
        <div
          role="button"
          tabIndex={leaving ? -1 : 0}
          data-notification-id={notification.id}
          data-glide-target
          aria-label={isUnread ? t("Unread: {0}", title) : title}
          className={cn(
            "group/row relative grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-2.5 rounded-md ui-focus-ring p-2.5 text-left outline-none",
            link || isUnread ? "cursor-pointer" : "cursor-default",
          )}
          onClick={activate}
          onKeyDown={onKeyDown}
        >
          {avatar ? (
            <ResolvedUserAvatar
              userId={avatar.userId}
              name={avatar.name}
              className="mt-px size-7"
              fallbackClassName="text-2xs"
              aria-hidden
            />
          ) : (
            <div
              className={cn(
                "mt-px flex size-7 items-center justify-center rounded-md",
                done ? "bg-surface-hover" : descriptor.tileClass,
              )}
              aria-hidden
            >
              <Icon
                className={cn("size-3.5", done ? "text-muted-foreground" : descriptor.iconClass)}
              />
            </div>
          )}

          <div className="min-w-0">
            <div className="flex items-baseline gap-2">
              <p
                className={cn(
                  "min-w-0 flex-1 text-sm text-pretty",
                  done ? "text-muted-foreground" : "font-medium text-foreground",
                )}
              >
                {title}
                {notification.occurrenceCount > 1 && (
                  <span
                    className="ml-1.5 inline-flex h-4.5 items-center rounded-badge bg-surface-active/70 px-1.5 align-baseline font-mono text-xs font-normal text-foreground tabular-nums"
                    aria-label={t(
                      "{0, plural, one {Came up # time} other {Came up # times}}",
                      notification.occurrenceCount,
                    )}
                  >
                    {notification.occurrenceCount}
                  </span>
                )}
              </p>
              {!later && (
                <span className="shrink-0 text-xs whitespace-nowrap text-foreground-subtle tabular-nums">
                  {formatRelativeTime(when - now, "narrow")}
                </span>
              )}
              {isUnread && (
                <span className="size-1.5 shrink-0 self-center rounded-full bg-brand" aria-hidden />
              )}
            </div>

            {showMessage && (
              <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
                {notification.message}
              </p>
            )}

            {!done && <NotificationContent notification={notification} />}

            {(critical || status !== null || later || note !== null) && (
              <div className="mt-1.5 flex flex-wrap items-center gap-1.5 text-xs text-foreground-subtle">
                {critical && (
                  <Badge variant="danger" dot className={STATE_BADGE}>
                    {t("Critical")}
                  </Badge>
                )}
                {status && (
                  <Badge variant={status.tone} dot className={STATE_BADGE}>
                    {t(status.label, ...(status.args ?? []))}
                  </Badge>
                )}
                {later && (
                  <Badge variant="neutral" className={STATE_BADGE}>
                    {comesBack(t, notification.surfacesAt, now)}
                  </Badge>
                )}
                {done && resolution !== null && (
                  <Badge variant="success" dot className={STATE_BADGE}>
                    {t("Resolved")}
                  </Badge>
                )}
                {note !== null && <span>{note}</span>}
              </div>
            )}

            {(actions.length > 0 || later || recent.length > 1) && (
              <div className="mt-2 flex flex-wrap items-center gap-1.5" onClick={stopPropagation}>
                {actions.map((action, index) => (
                  <RowAction
                    key={action.kind + action.label}
                    notification={notification}
                    action={action}
                    solid={index === 0 && (isWriteAction(action) || action.primary === true)}
                    handlers={handlers}
                  />
                ))}
                {later && (
                  <Button
                    type="button"
                    size="xs"
                    variant="secondary"
                    className="text-xs"
                    onClick={() => handlers.wake(notification.id)}
                  >
                    {t("Bring back now")}
                  </Button>
                )}
                {recent.length > 1 && (
                  <Button
                    type="button"
                    size="xs"
                    variant="quiet"
                    className="text-xs"
                    aria-expanded={expanded}
                    onClick={() => setExpanded((value) => !value)}
                  >
                    <ChevronRightIcon
                      className={cn("size-3 transition-transform", expanded && "rotate-90")}
                    />
                    {t("Latest {0}", recent.length)}
                  </Button>
                )}
              </div>
            )}

            {!done && !later && (
              <NotificationReply
                notification={notification}
                onReplied={() => handlers.done(notification.id)}
              />
            )}

            {expanded && recent.length > 1 && (
              <div onClick={stopPropagation}>
                <RecentOccurrences
                  notification={notification}
                  occurrences={recent}
                  now={now}
                  handlers={handlers}
                />
              </div>
            )}
          </div>

          <NotificationQuickActions
            notification={notification}
            lens={lens}
            handlers={handlers}
            laterMenu={laterMenu}
            moreMenu={moreMenu}
          />
        </div>
      </div>
    </div>
  );
}
