import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import { getNotificationDayGroup, type NotificationDayGroup } from "@/lib/notification-helpers";
import { settingsPath } from "@/lib/settings-path";
import { AlertCircleIcon, CheckDoubleIcon, Settings01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { createDropdownMenuHandle } from "@trenova/shared/components/ui/dropdown-menu";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  useDismissAllNotifications,
  useNotificationCounts,
  useNotificationFeed,
} from "@trenova/shared/hooks/use-notifications";
import { useNow } from "@trenova/shared/hooks/use-now";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Notification } from "@trenova/shared/types/notification";
import {
  useImperativeHandle,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
  type Ref,
} from "react";
import { useNavigate } from "react-router";
import { NotificationEmpty } from "./notification-empty";
import { NotificationItem } from "./notification-item";
import {
  lensCount,
  NOTIFICATION_LENS_LABELS,
  NOTIFICATION_PANEL_LENSES,
  type NotificationPanelLens,
} from "./notification-lenses";
import {
  laterTriggerId,
  NotificationLaterMenu,
  NotificationMoreMenu,
} from "./notification-row-menus";
import { NotificationFeedSkeleton } from "./notification-skeletons";
import { useNotificationRowHandlers } from "./use-notification-row-handlers";

const EMPTY_FEED: Notification[] = [];

/** A row before it is measured: a title, two lines of message, the why line and an action. */
const ROW_ESTIMATE = 96;
const FADE_HEIGHT = 28;

const GROUP_LABELS: Record<NotificationDayGroup | "later", string> = defineLabels({
  Today: "Today",
  Yesterday: "Yesterday",
  "This week": "This week",
  Older: "Older",
  later: "Coming back",
});

/** The heading a row sits under: the day it surfaced or was finished, or "Coming back". */
function groupOf(notification: Notification, lens: NotificationPanelLens) {
  if (lens === "later") return "later";

  return getNotificationDayGroup(
    lens === "done"
      ? (notification.dismissedAt ?? notification.surfacesAt)
      : notification.surfacesAt,
  );
}

const SHORTCUT_BLOCKERS = "input, textarea, select, [contenteditable='true'], [role='menu']";

/**
 * The panel's keyboard shortcuts, called for every key pressed anywhere in the sheet. The
 * sheet owns the element the keys arrive on (its close button and the sheet itself can hold
 * the focus), so it holds a ref to this and the panel fills it in.
 */
export type NotificationPanelKeyHandler = (event: KeyboardEvent<HTMLElement>) => void;

function Shortcut({ keys, children }: { keys: string[]; children: ReactNode }) {
  return (
    <span className="flex items-center gap-1">
      {keys.map((key) => (
        <Kbd key={key} className="h-4 min-w-4 border-0 bg-surface-hover px-1 font-mono">
          {key}
        </Kbd>
      ))}
      {children}
    </span>
  );
}

/**
 * The notification panel: what is the person's own, what went to their team, what they set
 * aside and what is finished. Each part loads when it is opened and mounts only the rows
 * near the viewport; the rows carry their own actions, and the keyboard works the list.
 */
export default function NotificationPanel({
  open,
  onClose,
  keys,
}: {
  open: boolean;
  onClose: () => void;
  keys: Ref<NotificationPanelKeyHandler>;
}) {
  const t = useT();
  const navigate = useNavigate();
  const now = useNow();

  const [lens, setLens] = useState<NotificationPanelLens>("forYou");
  const [laterMenu] = useState(() => createDropdownMenuHandle<Notification>());
  const [moreMenu] = useState(() => createDropdownMenuHandle<Notification>());
  const rootRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<VirtualRowsHandle>(null);

  const { data: counts } = useNotificationCounts(open);
  const {
    data: feed,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useNotificationFeed({ lens }, open);
  const dismissAll = useDismissAllNotifications();
  const { handlers, leaving } = useNotificationRowHandlers(onClose);

  const notifications = feed?.notifications ?? EMPTY_FEED;

  const rows: VirtualRow[] = [];
  let heading: string | null = null;
  for (const notification of notifications) {
    const group = groupOf(notification, lens);
    if (group !== heading) {
      heading = group;
      rows.push({
        key: `group:${group}`,
        render: () => (
          <p className="px-2.5 pt-3 pb-1 text-xs font-medium text-foreground-subtle">
            {t(GROUP_LABELS[group])}
          </p>
        ),
      });
    }
    rows.push({
      key: notification.id,
      render: () => (
        <NotificationItem
          notification={notification}
          lens={lens}
          now={now}
          leaving={leaving.has(notification.id)}
          handlers={handlers}
          laterMenu={laterMenu}
          moreMenu={moreMenu}
        />
      ),
    });
  }

  const rowElement = (id: string) =>
    rootRef.current?.querySelector<HTMLElement>(`[data-notification-id="${CSS.escape(id)}"]`) ??
    null;

  const focusRow = (id: string) => {
    listRef.current?.scrollToKey(id);
    requestAnimationFrame(() => rowElement(id)?.focus());
  };

  /** The row to carry on from when the one at `index` leaves: the next, or the one before. */
  const neighbourOf = (index: number) => notifications[index + 1] ?? notifications[index - 1];

  const nextRow = (id: string) => {
    const neighbour = neighbourOf(notifications.findIndex((item) => item.id === id));
    return neighbour ? rowElement(neighbour.id) : null;
  };

  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
    if (!(event.target instanceof HTMLElement) || event.target.closest(SHORTCUT_BLOCKERS)) return;

    const key = event.key.toLowerCase();
    const id = event.target.closest<HTMLElement>("[data-notification-id]")?.dataset.notificationId;
    const index = id === undefined ? -1 : notifications.findIndex((item) => item.id === id);

    if (key === "j" || key === "k") {
      const next = notifications[index === -1 ? 0 : index + (key === "j" ? 1 : -1)];
      if (next) {
        event.preventDefault();
        focusRow(next.id);
      }
      return;
    }
    if (index === -1 || id === undefined || lens === "done" || lens === "later") return;

    if (key === "e") {
      event.preventDefault();
      const neighbour = neighbourOf(index);
      handlers.done(id);
      if (neighbour) focusRow(neighbour.id);
    } else if (key === "h") {
      event.preventDefault();
      laterMenu.open(laterTriggerId(id));
    }
  };

  useImperativeHandle(keys, () => onKeyDown);

  const openSettings = () => {
    onClose();
    void navigate(settingsPath("notifications"));
  };

  const canMarkAllDone = (lens === "forYou" || lens === "team") && (counts?.[lens] ?? 0) > 0;

  return (
    <div ref={rootRef} className="flex min-h-0 flex-1 flex-col">
      <div className="flex flex-col gap-2 pt-3 pr-3 pl-4">
        <div className="flex min-h-7 items-center gap-2 pr-9">
          <SheetTitle className="min-w-0 flex-1 text-lg font-semibold">
            {t("Notifications")}
          </SheetTitle>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="quiet"
                  size="icon-sm"
                  aria-label={t("Notification settings")}
                  onClick={openSettings}
                />
              }
            >
              <Settings01Icon className="size-3.5" />
            </TooltipTrigger>
            <TooltipContent side="bottom">{t("Notification settings")}</TooltipContent>
          </Tooltip>
        </div>

        <div className="-mr-3 -ml-4 flex items-end gap-0.5 border-b border-border pr-3 pl-2">
          <Tabs
            value={lens}
            onValueChange={(value) => setLens(value as NotificationPanelLens)}
            className="gap-0"
          >
            <TabsList variant="underline" className="py-0">
              {NOTIFICATION_PANEL_LENSES.map((value) => {
                const count = lensCount(counts, value);

                return (
                  <TabsTab
                    key={value}
                    value={value}
                    className="group/tab h-8 gap-1.5 px-2 text-sm sm:h-8 sm:text-sm"
                  >
                    {t(NOTIFICATION_LENS_LABELS[value])}
                    {count !== null && (
                      <span className="font-mono text-xs font-normal text-foreground-subtle tabular-nums group-data-active/tab:text-muted-foreground">
                        {count}
                      </span>
                    )}
                  </TabsTab>
                );
              })}
            </TabsList>
          </Tabs>
          <span className="flex-1" />
          {canMarkAllDone && (
            <Button
              type="button"
              variant="quiet"
              size="xs"
              className="mb-1 px-2 text-xs"
              isLoading={dismissAll.isPending}
              onClick={() => dismissAll.mutate(lens)}
            >
              <CheckDoubleIcon className="size-3" />
              <span className="max-sm:sr-only">{t("Mark all done")}</span>
            </Button>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="min-h-0 flex-1 overflow-hidden">
          <NotificationFeedSkeleton />
        </div>
      ) : isError ? (
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 py-16">
          <AlertCircleIcon className="size-5 text-danger-foreground" />
          <p className="text-sm text-muted-foreground">{t("Notifications couldn't be loaded.")}</p>
          <Button type="button" size="xs" onClick={() => void refetch()}>
            {t("Try again")}
          </Button>
        </div>
      ) : (
        <VirtualRows
          rows={rows}
          handleRef={listRef}
          estimateSize={ROW_ESTIMATE}
          className="min-h-0 flex-1"
          contentClassName="px-1.5 pb-2.5"
          aria-label={t(NOTIFICATION_LENS_LABELS[lens])}
          hoverGlide
          glideClassName="bg-surface-hover"
          mask="overlay"
          maskHeight={FADE_HEIGHT}
          onEndReached={() => {
            if (hasNextPage && !isFetchingNextPage) void fetchNextPage();
          }}
          footer={
            isFetchingNextPage ? (
              <div className="flex items-center justify-center py-3">
                <Spinner className="size-3.5 text-muted-foreground" />
              </div>
            ) : null
          }
          empty={
            <div className="min-h-0 flex-1 overflow-y-auto">
              <NotificationEmpty lens={lens} />
            </div>
          }
        />
      )}

      <div className="flex items-center gap-3.5 border-t border-border px-4 py-1.5 text-xs text-foreground-subtle">
        <Shortcut keys={["↵"]}>{t("Open")}</Shortcut>
        <Shortcut keys={["E"]}>{t("Done")}</Shortcut>
        <span className="max-sm:hidden">
          <Shortcut keys={["H"]}>{t("Later")}</Shortcut>
        </span>
        <span className="flex-1" />
        <span className="max-sm:hidden">
          <Shortcut keys={["J", "K"]}>{t("Move")}</Shortcut>
        </span>
      </div>

      <NotificationLaterMenu handle={laterMenu} onLater={handlers.later} nextRow={nextRow} />
      <NotificationMoreMenu handle={moreMenu} handlers={handlers} />
    </div>
  );
}
