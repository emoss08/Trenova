import { useT } from "@trenova/shared/i18n/use-t";
import { useRecentActivityInfinite } from "@/hooks/use-attention";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { useSidebarQuickActions } from "@/hooks/use-sidebar-quick-actions";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { notification as notificationQueries } from "@trenova/shared/lib/queries/notification";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { Star01Icon } from "@trenova/shared/components/icons";
import { useMemo } from "react";
import { Link } from "react-router";
import { ActivityRow } from "@/components/navigation/activity-section";
import { WidgetEmpty, WidgetShell, WidgetSkeleton } from "../widget-shell";
import type { WidgetProps } from "../widget-registry";

export function QuickActionsWidget({ widget }: WidgetProps) {
  const t = useT();

  const actions = useSidebarQuickActions();

  return (
    <WidgetShell title={widget.title || t("Quick actions")}>
      {actions.length === 0 ? (
        <WidgetEmpty>{t("Pick your quick actions from the sidebar settings.")}</WidgetEmpty>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-x-2 gap-y-0.5">
          {actions.map(({ definition, icon: Icon, href }) => (
            <Link
              key={definition.id}
              to={href}
              title={t(definition.description)}
              className="flex min-w-0 items-center gap-2 rounded-md ui-focus-ring px-1.5 py-1.5 text-sm transition-colors outline-none hover:bg-surface-hover"
            >
              <Icon aria-hidden className="size-3.5 shrink-0 text-foreground-subtle" />
              <span className="truncate">{t(definition.label)}</span>
            </Link>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

export function FavoritesWidget({ widget }: WidgetProps) {
  const t = useT();

  const { data: favorites, isLoading } = useQuery(pageFavorite.all());

  return (
    <WidgetShell title={widget.title || t("Favorites")} count={favorites?.length}>
      {isLoading ? (
        <WidgetSkeleton />
      ) : !favorites || favorites.length === 0 ? (
        <WidgetEmpty>{t("Star a page to pin it here.")}</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-0.5">
          {favorites.map((favorite) => (
            <Link
              key={favorite.id}
              to={favorite.pageUrl}
              className="flex items-center gap-2 rounded-md ui-focus-ring px-1.5 py-1.5 text-sm transition-colors outline-none hover:bg-surface-hover"
            >
              <Star01Icon
                aria-hidden
                className="size-3.5 shrink-0 fill-warning text-warning-foreground"
              />
              <span className="truncate">{favorite.pageTitle}</span>
            </Link>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

/**
 * Recent activity, read two ways. The activity widget is the organization's
 * stream; jump-back-in narrows the same source to what this user touched, which
 * is the difference between "what happened" and "where was I".
 */
function ActivityRows({ limit, mineOnly }: { limit: number; mineOnly: boolean }) {
  const t = useT();

  // Ask for more than we show when filtering to this user: the stream is
  // organization-wide, so a page of it may hold only a few of their entries.
  const { data: entries, isLoading } = useRecentActivityInfinite(mineOnly ? limit * 3 : limit);
  const currentUserId = useAuthStore((state) => state.user?.id);

  const rows = useMemo(() => {
    const all = entries ?? [];
    const filtered = mineOnly ? all.filter((entry) => entry.user?.id === currentUserId) : all;
    return filtered.slice(0, limit);
  }, [entries, mineOnly, currentUserId, limit]);

  if (isLoading) return <WidgetSkeleton rows={4} />;
  if (rows.length === 0) {
    return (
      <WidgetEmpty>
        {mineOnly ? t("Nothing to jump back into yet.") : t("No recent activity.")}
      </WidgetEmpty>
    );
  }

  return (
    <div className="-mx-1 flex flex-col gap-0.5">
      {rows.map((entry) => (
        <ActivityRow key={entry.id} entry={entry} />
      ))}
    </div>
  );
}

export function ActivityWidget({ widget }: WidgetProps) {
  const t = useT();

  return (
    <WidgetShell
      title={widget.title || t("Activity")}
      href="/admin/audit-logs"
      hrefLabel={t("Open audit log")}
    >
      <ActivityRows limit={widget.config.limit ?? 10} mineOnly={false} />
    </WidgetShell>
  );
}

export function JumpBackInWidget({ widget }: WidgetProps) {
  const t = useT();

  return (
    <WidgetShell title={widget.title || t("Jump back in")}>
      <ActivityRows limit={widget.config.limit ?? 8} mineOnly />
    </WidgetShell>
  );
}

/**
 * Saved views live in two places — table configurations and report views — and
 * a user thinks of both as "the views I set up", so this points at each rather
 * than pretending one list exists.
 */
export function SavedViewsWidget({ widget }: WidgetProps) {
  const t = useT();

  const targets = [
    { label: t("Shipment board views"), href: "/shipment-management/shipments" },
    { label: t("Report views"), href: "/reports" },
    { label: t("Dashboards"), href: "/reports/dashboards" },
  ];

  return (
    <WidgetShell title={widget.title || t("Saved views")}>
      <div className="flex flex-col gap-0.5">
        {targets.map((target) => (
          <Link
            key={target.href}
            to={target.href}
            className="rounded-md ui-focus-ring px-1.5 py-1.5 text-sm transition-colors outline-none hover:bg-surface-hover"
          >
            {target.label}
          </Link>
        ))}
      </div>
    </WidgetShell>
  );
}

export function NotificationsWidget({ widget }: WidgetProps) {
  const t = useT();

  const limit = widget.config.limit ?? 8;
  const { data: notifications, isLoading } = useQuery({
    ...notificationQueries.feed({ first: limit, lens: "forYou", unreadOnly: true }),
    select: (data) => data.results,
  });
  // The badge is the true count of what is the person's own and unread, not the page size
  // the list happened to fetch — a widget showing 8 rows over 40 unread must not read as "8".
  const { data: unreadCount } = useUnreadNotificationCount();

  return (
    <WidgetShell
      title={widget.title || t("Notifications")}
      count={unreadCount ?? notifications?.length}
    >
      {isLoading ? (
        <WidgetSkeleton />
      ) : !notifications || notifications.length === 0 ? (
        <WidgetEmpty>{t("You’re all caught up.")}</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-0.5">
          {notifications.map((notification) => (
            <div key={notification.id} className="flex flex-col gap-0.5 rounded-md px-1.5 py-1">
              <span className="truncate text-xs font-medium">{t(notification.title)}</span>
              <span className="truncate text-2xs text-muted-foreground">
                {notification.message}
              </span>
            </div>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}
