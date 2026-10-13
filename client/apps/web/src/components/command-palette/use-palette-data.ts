import { ATTENTION_ROWS, type AttentionRowConfig } from "@/config/attention-rows";
import { navigationConfig } from "@/config/navigation.config";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { useAttentionSummary } from "@/hooks/use-attention";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { apiService } from "@/services/api";
import { useRecentRecords } from "@/stores/recent-records-store";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { useUnreadNotificationCount } from "@trenova/shared/hooks/use-notifications";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { isMacPlatform } from "@trenova/shared/lib/shortcuts";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { queryOptions, useQuery } from "@tanstack/react-query";
import { Pin01Icon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { useLocation } from "react-router";
import type { ActionContext } from "./palette-actions";
import { buildAppCommands, buildQuickActionCommands } from "./palette-commands";
import type {
  PaletteAttention,
  PaletteAttentionTone,
  PaletteCommand,
  PaletteIcon,
  PalettePage,
  PaletteScope,
} from "./palette-model";
import { isRecordScope } from "./palette-model";
import {
  MIN_REMOTE_QUERY_LENGTH,
  type HomeInput,
  type RecentPaletteRecord,
  type RemoteState,
} from "./palette-sections";
import { buildRouteCommandGroups } from "./route-command-data";

const QUICK_ACTION_IDS = [
  "quick:create-shipment",
  "app:assistant-new",
  "quick:open-insights",
  "quick:create-customer",
];
const ALL_SCOPE_LIMIT = 5;
const RECORD_SCOPE_LIMIT = 25;
const REMOTE_STALE_TIME = 15_000;

const ATTENTION_TONE: Record<AttentionRowConfig["tone"], PaletteAttentionTone> = {
  default: "neutral",
  warning: "warning",
  destructive: "danger",
};

/** What each attention count holds, said once for the preview. */
const ATTENTION_REASONS = defineLabels({
  billingQueue: "Delivered shipments waiting to be invoiced.",
  pendingApprovals: "Invoice adjustments waiting on an approver.",
  reconciliationExceptions: "Adjustments that could not be reconciled against their invoices.",
  serviceFailures: "Late or missed stops with no resolution recorded.",
  ediAttention: "EDI documents that failed or are waiting on a reply.",
  agentDecisions: "Changes your agents proposed that need a decision.",
});

function normalizeHref(href: string): string {
  const trimmed = href.replace(/\/+$/, "");
  return trimmed === "" ? "/" : trimmed;
}

function pathOf(href: string): string {
  return normalizeHref(href.split("?")[0] ?? href);
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
    trail: [],
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
  quickActions: PaletteCommand[];
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
  const canCreateShipment = hasPermission(Resource.Shipment, Operation.Create);
  const currentHref = `${pathname}${search}${hash}`;

  const pages = useMemo(
    () =>
      buildRouteCommandGroups(filteredModules, adminLinks).flatMap((group) =>
        group.items.map((item): PalettePage => ({
          id: item.id,
          title: item.title,
          trail: item.trail,
          href: item.href,
          icon: item.icon,
          keywords: item.keywords,
          module: group.label,
          glance: item.glance,
          recordType: item.recordType,
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

  const quickActions = useMemo(() => {
    const byId = new Map(commands.map((command) => [command.id, command]));
    return QUICK_ACTION_IDS.flatMap((id) => {
      const command = byId.get(id);
      return command ? [command] : [];
    });
  }, [commands]);

  const relatedCommands = useCallback(
    (href: string) => {
      const path = pathOf(href);
      return commands.filter(
        (command) =>
          command.group === "create" &&
          command.intent.type === "navigate" &&
          pathOf(command.intent.href) === path,
      );
    },
    [commands],
  );

  const actionContext = useMemo(
    () => ({
      mac,
      canUseAssistant,
      canCreateShipment,
      canReach: (path: string) => pageIndex.has(pathOf(path)),
      relatedCommands,
    }),
    [canCreateShipment, canUseAssistant, mac, pageIndex, relatedCommands],
  );

  return { pages, pageIndex, commands, quickActions, actionContext };
}

export function useUnreadCount(): number {
  const { data } = useUnreadNotificationCount();
  return data ?? 0;
}

export function useRecentPaletteRecords(): readonly RecentPaletteRecord[] {
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recent = useRecentRecords(organizationId);
  return useMemo(
    () =>
      recent.map((entry) => ({
        record: {
          entityType: entry.entityType,
          id: entry.id,
          title: entry.title,
          subtitle: entry.subtitle,
          href: entry.href,
          metadata: entry.metadata ?? {},
        },
        visitedAt: entry.visitedAt,
      })),
    [recent],
  );
}

export function usePinnedPages(open: boolean, pageIndex: ReadonlyMap<string, PalettePage>) {
  const { data, isLoading } = useQuery({ ...pageFavorite.all(), enabled: open });
  const pages = useMemo(
    () =>
      (data ?? []).map((favorite) =>
        pageFromHref(favorite.pageUrl, favorite.pageTitle, pageIndex, Pin01Icon),
      ),
    [data, pageIndex],
  );
  const pinnedUrls = useMemo(() => new Set(pages.map((page) => page.href)), [pages]);
  return { pages, pinnedUrls, loading: isLoading };
}

/**
 * The counts that need someone, as rows. A count the summary could not read
 * (no permission, or the section failed) is left out rather than shown as 0.
 */
export function usePaletteAttention(): { rows: PaletteAttention[]; loading: boolean } {
  const t = useT();
  const { data: summary, isLoading } = useAttentionSummary();
  const rows = useMemo(
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
            reason: Object.hasOwn(ATTENTION_REASONS, row.key)
              ? ATTENTION_REASONS[row.key as keyof typeof ATTENTION_REASONS]
              : "",
          },
        ];
      }),
    [summary, t],
  );
  return { rows, loading: isLoading };
}

/**
 * The live half of the empty palette: what was opened last, the counts that
 * need someone, and pinned pages. Nothing here is fetched until the palette
 * is open, and every query keeps its own loading state.
 */
export function usePaletteHome({
  catalog,
  pinned,
}: {
  catalog: PaletteCatalog;
  pinned: ReturnType<typeof usePinnedPages>;
}): HomeInput {
  const recentRecords = useRecentPaletteRecords();
  const attention = usePaletteAttention();

  return useMemo(
    () => ({
      recentRecords,
      attention,
      pinnedPages: { pages: pinned.pages, loading: pinned.loading },
      quickActions: catalog.quickActions,
    }),
    [attention, catalog.quickActions, pinned.loading, pinned.pages, recentRecords],
  );
}

/**
 * The global index searched for one palette query, in one scope. The list and
 * a question asked from it read the same cache, so asking right after the
 * records appear sends what was drawn rather than searching again.
 */
export function isRemoteSearchable(query: string, scope: PaletteScope): boolean {
  const trimmed = query.trim();
  const recordScope = isRecordScope(scope);
  if (scope !== "all" && !recordScope) {
    return false;
  }
  return (recordScope && trimmed === "") || trimmed.length >= MIN_REMOTE_QUERY_LENGTH;
}

export function paletteSearchQuery(query: string, scope: PaletteScope) {
  const trimmed = query.trim();
  const recordScope = isRecordScope(scope);
  const entityTypes = recordScope ? [scope] : undefined;
  const limit = recordScope ? RECORD_SCOPE_LIMIT : ALL_SCOPE_LIMIT;

  return queryOptions({
    queryKey: ["global-search", trimmed, scope, limit, entityTypes] as const,
    queryFn: ({ signal }) =>
      apiService.globalSearchService.search(trimmed, limit, entityTypes, { signal }),
    staleTime: REMOTE_STALE_TIME,
  });
}

/**
 * Record search against the global index. A superseded query is cancelled
 * rather than left to land late, and what was last found stays on screen
 * while the next query is in flight so the list does not blink per key. A
 * record scope with nothing typed browses that kind of record.
 */
export function usePaletteRemoteSearch({
  enabled,
  query,
  scope,
}: {
  /** Off while the palette is closed or asking about one record. */
  enabled: boolean;
  /** What to search records for: what was typed, or a question's search words. */
  query: string;
  scope: PaletteScope;
}): RemoteState & { retry: () => void } {
  const trimmed = query.trim();
  const ready = enabled && isRemoteSearchable(trimmed, scope);

  const result = useQuery({
    ...paletteSearchQuery(trimmed, scope),
    enabled: ready,
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[2] === scope && previousQuery.queryKey[1] !== "" && trimmed !== ""
        ? previous
        : undefined,
  });

  return {
    groups: ready ? (result.data?.groups ?? []) : [],
    loading: ready && result.isFetching,
    error: ready && result.isError,
    ready,
    retry: () => void result.refetch(),
  };
}
