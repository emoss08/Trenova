import { buildCommandHref } from "@/components/command-palette/route-command-data";
import { navigationConfig } from "@/config/navigation.config";
import type { QuickActionCommand } from "@/config/navigation.types";
import { QUICK_ACTION_ICONS } from "@/config/quick-action-icons";
import { useRecentActivityInfinite } from "@/hooks/use-attention";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { queries } from "@/lib/queries";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { StarIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { WidgetCount, WidgetEmpty, WidgetShell, WidgetSkeleton } from "../widget-shell";
import type { WidgetProps } from "../widget-registry";

export function QuickActionsWidget({ widget }: WidgetProps) {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const { data: preferences } = useSidebarPreferences();

  const actions = useMemo(() => {
    const definitions = navigationConfig.quickActions ?? [];
    const definitionsById = new Map(definitions.map((definition) => [definition.id, definition]));

    return (preferences?.quickActionIds ?? [])
      .map((id) => definitionsById.get(id))
      .filter((definition): definition is QuickActionCommand => {
        if (!definition || !QUICK_ACTION_ICONS[definition.id]) return false;
        if (!definition.resource) return true;
        return hasPermission(definition.resource, definition.requiredOperation ?? Operation.Create);
      })
      .map((definition) => ({
        definition,
        icon: QUICK_ACTION_ICONS[definition.id],
        href: buildCommandHref(definition.path, definition.query),
      }));
  }, [hasPermission, preferences?.quickActionIds]);

  return (
    <WidgetShell title={widget.title || "Quick Actions"}>
      {actions.length === 0 ? (
        <WidgetEmpty>Pick your quick actions from the sidebar settings.</WidgetEmpty>
      ) : (
        <div className="grid grid-cols-2 gap-1.5">
          {actions.map(({ definition, icon: Icon, href }) => (
            <Link
              key={definition.id}
              to={href}
              title={definition.description}
              className="flex h-7 items-center gap-1.5 truncate rounded-md border border-border bg-background px-2 text-xs font-medium text-foreground/80 transition-colors hover:bg-muted hover:text-foreground"
            >
              <Icon className="size-3 shrink-0" />
              <span className="truncate">{definition.label}</span>
            </Link>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

export function FavoritesWidget({ widget }: WidgetProps) {
  const { data: favorites, isLoading } = useQuery(queries.pageFavorite.all());

  return (
    <WidgetShell
      title={widget.title || "Favorites"}
      badge={<WidgetCount value={favorites?.length} />}
    >
      {isLoading ? (
        <WidgetSkeleton />
      ) : !favorites || favorites.length === 0 ? (
        <WidgetEmpty>Star a page to pin it here.</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-0.5">
          {favorites.map((favorite) => (
            <Link
              key={favorite.id}
              to={favorite.pageUrl}
              className="flex items-center gap-2 rounded px-1.5 py-1 text-xs transition-colors hover:bg-muted/60"
            >
              <StarIcon className="size-3 shrink-0 fill-amber-400 text-amber-400" />
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
        {mineOnly ? "Nothing to jump back into yet." : "No recent activity."}
      </WidgetEmpty>
    );
  }

  return (
    <div className="flex flex-col gap-0.5">
      {rows.map((entry) => (
        <div key={entry.id} className="flex flex-col gap-0.5 rounded px-1.5 py-1">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate text-xs">
              <span className="font-medium">{entry.user?.name ?? "Someone"}</span>{" "}
              <span className="text-muted-foreground">{entry.operation}</span>{" "}
              <span className="text-muted-foreground">{entry.resource}</span>
            </span>
            <span className="shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
              {formatToUserTimezone(entry.timestamp, {
                showTimeZone: false,
                showSeconds: false,
              })}
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}

export function ActivityWidget({ widget }: WidgetProps) {
  return (
    <WidgetShell
      title={widget.title || "Activity"}
      href="/admin/audit-logs"
      hrefLabel="Open audit log"
    >
      <ActivityRows limit={widget.config.limit ?? 10} mineOnly={false} />
    </WidgetShell>
  );
}

export function JumpBackInWidget({ widget }: WidgetProps) {
  return (
    <WidgetShell title={widget.title || "Jump Back In"}>
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
  const targets = [
    { label: "Shipment board views", href: "/shipment-management/shipments" },
    { label: "Report views", href: "/reports" },
    { label: "Dashboards", href: "/reports/dashboards" },
  ];

  return (
    <WidgetShell title={widget.title || "Saved Views"}>
      <div className="flex flex-col gap-0.5">
        {targets.map((target) => (
          <Link
            key={target.href}
            to={target.href}
            className="rounded px-1.5 py-1 text-xs transition-colors hover:bg-muted/60"
          >
            {target.label}
          </Link>
        ))}
      </div>
    </WidgetShell>
  );
}

export function NotificationsWidget({ widget }: WidgetProps) {
  const limit = widget.config.limit ?? 8;
  const { data: notifications, isLoading } = useQuery({
    ...queries.notification.feed({ first: limit, unreadOnly: true }),
    select: (data) => data.results,
  });
  // The badge is the true unread total, not the page size the list happened to
  // fetch — a widget showing 8 rows over 40 unread must not read as "8".
  const { data: unreadCount } = useUnreadNotificationCount();

  return (
    <WidgetShell
      title={widget.title || "Notifications"}
      badge={<WidgetCount value={unreadCount ?? notifications?.length} />}
    >
      {isLoading ? (
        <WidgetSkeleton />
      ) : !notifications || notifications.length === 0 ? (
        <WidgetEmpty>You&rsquo;re all caught up.</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-0.5">
          {notifications.map((notification) => (
            <div key={notification.id} className="flex flex-col gap-0.5 rounded px-1.5 py-1">
              <span className="truncate text-xs font-medium">{notification.title}</span>
              <span className="truncate text-[10px] text-muted-foreground">
                {notification.message}
              </span>
            </div>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}
