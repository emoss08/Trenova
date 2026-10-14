import { useT } from "@trenova/shared/i18n/use-t";
import { ActivityFeed } from "@/components/navigation/activity-section";
import { CustomizeSidebarDialog } from "@/components/navigation/customize-sidebar-dialog";
import { AttentionCountBadge } from "@/components/navigation/sidebar-chrome";
import {
  WorkspaceGroupLabel,
  WorkspaceNavRow,
  WorkspaceRowLabel,
} from "@/components/navigation/workspace-primitives";
import { useAttentionRows } from "@/hooks/use-attention";
import { SIDEBAR_SECTION_KEYS } from "@/lib/graphql/sidebar-preferences";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { isRouteActive } from "@/lib/route-utils";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useQuery } from "@tanstack/react-query";
import { Star01Icon } from "@trenova/shared/components/icons";
import { useLocation } from "react-router";

function AttentionRows() {
  const t = useT();

  const { pathname } = useLocation();
  const { rows, isLoading } = useAttentionRows();

  if (isLoading) {
    return (
      <div className="flex flex-col gap-1 px-2 pt-3">
        {Array.from({ length: 3 }, (_, index) => (
          <Skeleton key={index} className="h-6.5 w-full rounded-md" />
        ))}
      </div>
    );
  }
  if (rows.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-0.5 px-2">
      <WorkspaceGroupLabel>{t("Needs attention")}</WorkspaceGroupLabel>
      {rows.map(({ row, count }) => (
        <WorkspaceNavRow key={row.key} to={row.path} active={isRouteActive(pathname, row.path)} sub>
          <WorkspaceRowLabel>{t(row.label)}</WorkspaceRowLabel>
          <AttentionCountBadge attention={count > 0 ? { count, tone: row.tone } : undefined} />
        </WorkspaceNavRow>
      ))}
    </div>
  );
}

function PinnedRows() {
  const t = useT();

  const { pathname } = useLocation();
  const { data: favorites } = useQuery(pageFavorite.all());

  if (!favorites || favorites.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-0.5 px-2">
      <WorkspaceGroupLabel>{t("Pinned")}</WorkspaceGroupLabel>
      {favorites.map((favorite) => (
        <WorkspaceNavRow
          key={favorite.id}
          to={favorite.pageUrl}
          active={isRouteActive(pathname, favorite.pageUrl)}
          sub
        >
          <Star01Icon className="size-3 shrink-0 fill-warning text-warning-foreground" />
          <WorkspaceRowLabel>{favorite.pageTitle}</WorkspaceRowLabel>
        </WorkspaceNavRow>
      ))}
    </div>
  );
}

/**
 * What the panel holds on Home: the way back to it, then the work that
 * needs you, the pages you pinned and what happened lately, each only when
 * you have not hidden it. The customize control lives here because this is
 * where its choices show.
 */
export function HomePanel({ hiddenSections }: { hiddenSections: ReadonlySet<string> }) {
  const t = useT();

  const { pathname } = useLocation();

  return (
    <>
      <div className="flex items-center gap-1 px-2 pt-2">
        <div className="min-w-0 flex-1">
          <WorkspaceNavRow to="/" active={pathname === "/"}>
            <WorkspaceRowLabel>{t("Home")}</WorkspaceRowLabel>
          </WorkspaceNavRow>
        </div>
        <CustomizeSidebarDialog ghost />
      </div>
      {!hiddenSections.has(SIDEBAR_SECTION_KEYS.attention) && <AttentionRows />}
      {!hiddenSections.has(SIDEBAR_SECTION_KEYS.favorites) && <PinnedRows />}
      {!hiddenSections.has(SIDEBAR_SECTION_KEYS.activity) && (
        <ActivityFeed
          emptyState="hidden"
          maxHeightClassName="max-h-64"
          heading={
            <div className="px-2">
              <WorkspaceGroupLabel>{t("Recent activity")}</WorkspaceGroupLabel>
            </div>
          }
        />
      )}
    </>
  );
}
