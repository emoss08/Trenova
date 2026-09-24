import { ATTENTION_ROWS } from "@/config/attention-rows";
import { navigationConfig } from "@/config/navigation.config";
import { getNotificationLink } from "@/components/notification-center/notification-registry";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { useAttentionSummary } from "@/hooks/use-attention";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useRecentPages } from "@/stores/recent-pages-store";
import { useRecentRecords } from "@/stores/recent-records-store";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { useT } from "@trenova/shared/i18n/use-t";
import { notification as notificationQueries } from "@trenova/shared/lib/queries/notification";
import { isMacPlatform } from "@trenova/shared/lib/shortcuts";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { HistoryIcon, PinIcon } from "lucide-react";
import { useMemo } from "react";
import { useLocation } from "react-router";
import type { ActionContext } from "./palette-actions";
import { buildAppCommands, buildQuickActionCommands } from "./palette-commands";
import type {
  PaletteAttention,
  PaletteCommand,
  PaletteIcon,
  PalettePage,
  PaletteRecord,
  PaletteScope,
} from "./palette-model";
import { isRecordScope } from "./palette-model";
import { MIN_REMOTE_QUERY_LENGTH, type HomeInput, type RemoteState } from "./palette-sections";
import { buildRouteCommandGroups } from "./route-command-data";

const SUGGESTED_COMMAND_IDS = [
  "quick:create-shipment",
  "quick:create-customer",
  "quick:create-worker",
  "quick:create-location",
  "quick:open-assistant",
  "app:shortcuts",
];
const HOME_NOTIFICATIONS = 3;
const ALL_SCOPE_LIMIT = 5;
const RECORD_SCOPE_LIMIT = 25;
const REMOTE_STALE_TIME = 15_000;

const ATTENTION_TONE: Record<(typeof ATTENTION_ROWS)[number]["tone"], PaletteAttention["tone"]> = {
  default: "info",
  warning: "warning",
  destructive: "danger",
};

function normalizeHref(href: string): string {
  const trimmed = href.replace(/\/+$/, "");
  return trimmed === "" ? "/" : trimmed;
}

function pageFromHref(
  href: string,
  title: string,
  index: ReadonlyMap<string, PalettePage>,
  fallbackIcon: PaletteIcon,
): PalettePage {
  const known = index.get(normalizeHref(href));
  if (known) {
    return { ...known, title: title || known.title, href };
  }
  return {
    id: href,
    title,
    trail: href,
    href,
    icon: fallbackIcon,
    keywords: [],
    module: "",
  };
}

export interface PaletteCatalog {
  pages: PalettePage[];
  pageIndex: ReadonlyMap<string, PalettePage>;
  commands: PaletteCommand[];
  suggested: PaletteCommand[];
  actionContext: Omit<ActionContext, "pinnedUrls">;
}

/**
 * Every page and command this person may reach, from the same filtered
 * navigation the sidebar draws: a page the sidebar hides, the palette does
 * not offer either.
 */
export function usePaletteCatalog(unreadNotifications: number): PaletteCatalog {
  const t = useT();
  const { pathname, search, hash } = useLocation();
  const filteredModules = useFilteredNavigation();
  const adminLinks = useAccessibleAdminLinks();
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const capabilities = useOrgCapabilities();
  const { theme } = useTheme();
  const mac = isMacPlatform();
  const canUseAssistant = hasPermission(Resource.Assistant, Operation.Read);
  const currentHref = `${pathname}${search}${hash}`;

  const pages = useMemo(
    () =>
      buildRouteCommandGroups(filteredModules, adminLinks).flatMap((group) =>
        group.items.map((item): PalettePage => ({
          id: item.id,
          title: item.title,
          trail: item.subtitle,
          href: item.href,
          icon: item.icon,
          keywords: item.keywords,
          module: group.label,
        })),
      ),
    [adminLinks, filteredModules],
  );

  const pageIndex = useMemo(
    () => new Map(pages.map((page) => [normalizeHref(page.href), page])),
    [pages],
  );

  const quickCommands = useMemo(
    () =>
      buildQuickActionCommands(
        navigationConfig.quickActions ?? [],
        (path) => pageIndex.get(normalizeHref(path))?.icon,
        { hasPermission, capabilities },
        t,
      ),
    [capabilities, hasPermission, pageIndex, t],
  );

  const appCommands = useMemo(
    () => buildAppCommands({ theme, unreadNotifications, canUseAssistant, currentHref, mac }, t),
    [canUseAssistant, currentHref, mac, t, theme, unreadNotifications],
  );

  const commands = useMemo(() => [...quickCommands, ...appCommands], [appCommands, quickCommands]);

  const suggested = useMemo(() => {
    const byId = new Map(commands.map((command) => [command.id, command]));
    return SUGGESTED_COMMAND_IDS.flatMap((id) => {
      const command = byId.get(id);
      return command ? [command] : [];
    });
  }, [commands]);

  const actionContext = useMemo(
    () => ({
      mac,
      canUseAssistant,
      canReach: (path: string) => pageIndex.has(normalizeHref(path)),
    }),
    [canUseAssistant, mac, pageIndex],
  );

  return { pages, pageIndex, commands, suggested, actionContext };
}

export function useUnreadCount(): number {
  const { data } = useUnreadNotificationCount();
  return data ?? 0;
}

export function useRecentPaletteRecords(): readonly PaletteRecord[] {
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recent = useRecentRecords(organizationId);
  return useMemo(
    () =>
      recent.map((record) => ({
        entityType: record.entityType,
        id: record.id,
        title: record.title,
        subtitle: record.subtitle,
        href: record.href,
        metadata: record.metadata ?? {},
      })),
    [recent],
  );
}

export function usePinnedPages(open: boolean, pageIndex: ReadonlyMap<string, PalettePage>) {
  const { data, isLoading } = useQuery({ ...queries.pageFavorite.all(), enabled: open });
  const pages = useMemo(
    () =>
      (data ?? []).map((favorite) =>
        pageFromHref(favorite.pageUrl, favorite.pageTitle, pageIndex, PinIcon),
      ),
    [data, pageIndex],
  );
  const pinnedUrls = useMemo(() => new Set(pages.map((page) => page.href)), [pages]);
  return { pages, pinnedUrls, loading: isLoading };
}

/**
 * The live half of the empty palette: counts that need someone, unread
 * notifications, pinned and recent pages. Nothing here is fetched until the
 * palette is open, and every query keeps its own loading state.
 */
export function usePaletteHome({
  open,
  catalog,
  pinned,
  unread,
}: {
  open: boolean;
  catalog: PaletteCatalog;
  pinned: ReturnType<typeof usePinnedPages>;
  unread: number;
}): HomeInput {
  const t = useT();
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recentRecords = useRecentPaletteRecords();
  const recentPageEntries = useRecentPages(organizationId);
  const { data: summary, isLoading: attentionLoading } = useAttentionSummary();

  const notifications = useQuery({
    ...notificationQueries.feed({ first: HOME_NOTIFICATIONS, unreadOnly: true }),
    enabled: open && unread > 0,
    staleTime: 10_000,
  });

  const attentionRows = useMemo(
    () =>
      ATTENTION_ROWS.flatMap((row): PaletteAttention[] => {
        const count = summary?.[row.key];
        if (typeof count !== "number" || count <= 0) {
          return [];
        }
        return [
          {
            key: row.key,
            label: t(row.label),
            module: t(row.module),
            count,
            href: row.path,
            tone: ATTENTION_TONE[row.tone],
          },
        ];
      }),
    [summary, t],
  );

  const notificationItems = useMemo(
    () =>
      (notifications.data?.results ?? []).map((notification) => ({
        notification,
        href: getNotificationLink(notification),
      })),
    [notifications.data?.results],
  );

  const recentPages = useMemo(
    () =>
      recentPageEntries.map((entry) =>
        pageFromHref(entry.path, entry.title, catalog.pageIndex, HistoryIcon),
      ),
    [catalog.pageIndex, recentPageEntries],
  );

  return {
    recentRecords,
    attention: { rows: attentionRows, loading: attentionLoading },
    notifications: {
      items: notificationItems,
      unread,
      loading: unread > 0 && notifications.isLoading,
    },
    pinnedPages: { pages: pinned.pages, loading: pinned.loading },
    recentPages,
    suggested: catalog.suggested,
  };
}

/**
 * Record search against the global index. A superseded query is cancelled
 * rather than left to land late, and what was last found stays on screen
 * while the next query is in flight so the list does not blink per key.
 */
export function usePaletteRemoteSearch({
  enabled,
  query,
  scope,
}: {
  /** Off while the palette is closed, listing a row's actions, or taking a question. */
  enabled: boolean;
  query: string;
  scope: PaletteScope;
}): RemoteState & { retry: () => void } {
  const trimmed = query.trim();
  const searchable = scope === "all" || isRecordScope(scope);
  const ready = enabled && searchable && trimmed.length >= MIN_REMOTE_QUERY_LENGTH;
  const entityTypes = isRecordScope(scope) ? [scope] : undefined;
  const limit = isRecordScope(scope) ? RECORD_SCOPE_LIMIT : ALL_SCOPE_LIMIT;

  const result = useQuery({
    queryKey: ["global-search", trimmed, scope, limit, entityTypes],
    queryFn: ({ signal }) =>
      apiService.globalSearchService.search(trimmed, limit, entityTypes, { signal }),
    enabled: ready,
    staleTime: REMOTE_STALE_TIME,
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[2] === scope ? previous : undefined,
  });

  return {
    groups: result.data?.groups ?? [],
    loading: result.isFetching,
    error: result.isError,
    ready,
    retry: () => void result.refetch(),
  };
}
