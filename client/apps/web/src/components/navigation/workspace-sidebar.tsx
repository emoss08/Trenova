import { useT } from "@trenova/shared/i18n/use-t";
import { ActivityFeed } from "@/components/navigation/activity-section";
import { useSidebarFold } from "@/components/navigation/sidebar-fold";
import { AttentionCountBadge, ModulePageList } from "@/components/navigation/sidebar-chrome";
import {
  moduleSidebarView,
  type ModuleAttention,
  type SidebarModuleView,
} from "@/components/navigation/sidebar-model";
import {
  WORKSPACE_SIDEBAR_WIDTH_CLASS,
  WorkspaceGroupLabel,
  WorkspaceNavRow,
  WorkspaceRowLabel,
} from "@/components/navigation/workspace-primitives";
import { useAttentionRows } from "@/hooks/use-attention";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { SIDEBAR_SECTION_KEYS } from "@/lib/graphql/sidebar-preferences";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { isRouteActive } from "@/lib/route-utils";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigationStore } from "@/stores/navigation-store";
import { Sheet, SheetContent, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { useQuery } from "@tanstack/react-query";
import { Star01Icon } from "@trenova/shared/components/icons";
import { useLocation } from "react-router";

function ModulePanel({
  view,
  activePath,
  attentionByPath,
}: {
  view: SidebarModuleView;
  activePath: string | null;
  attentionByPath: ReadonlyMap<string, ModuleAttention>;
}) {
  return (
    <ModulePageList
      view={view}
      activePath={activePath}
      attentionByPath={attentionByPath}
      className="px-2 pt-2"
    />
  );
}

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

function HomePanel({ hiddenSections }: { hiddenSections: ReadonlySet<string> }) {
  const t = useT();

  const { pathname } = useLocation();

  return (
    <>
      <div className="px-2 pt-2">
        <WorkspaceNavRow to="/" active={pathname === "/"}>
          <WorkspaceRowLabel>{t("Home")}</WorkspaceRowLabel>
        </WorkspaceNavRow>
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

function WorkspaceSidebarBody() {
  const { home, views, activeModule, activePath, pageAttention, hiddenSections } =
    useSidebarNavigation();

  const activeView = moduleSidebarView(activeModule, views);

  return (
    <ScrollArea
      className={cn("min-h-0 flex-1", WORKSPACE_SIDEBAR_WIDTH_CLASS)}
      maskHeight={16}
      maskVariant="sidebar"
    >
      {activeView ? (
        <ModulePanel view={activeView} activePath={activePath} attentionByPath={pageAttention} />
      ) : home ? (
        <HomePanel hiddenSections={hiddenSections} />
      ) : null}
      <div className="h-2" aria-hidden />
    </ScrollArea>
  );
}

/**
 * The sidebar of the workspace layout. It never lists the product; it lists
 * the module you are in, and Home lists what needs you. Hidden, it takes no
 * room and no focus; the toggle lives in the context bar. On a phone it never
 * sits beside the page: the toggle slides it over the page as a sheet.
 */
export function WorkspaceSidebar() {
  const t = useT();

  const { compact, shown } = useSidebarFold();
  const setCompactSidebarOpen = useNavigationStore((state) => state.setCompactSidebarOpen);

  if (compact) {
    return (
      <Sheet open={shown} onOpenChange={setCompactSidebarOpen}>
        <SheetContent
          side="left"
          showCloseButton={false}
          className="bg-sidebar w-auto gap-0 rounded-none border-y-0 border-l-0 p-0"
        >
          <SheetTitle className="sr-only">{t("Sidebar")}</SheetTitle>
          <WorkspaceSidebarBody />
        </SheetContent>
      </Sheet>
    );
  }

  return (
    <aside
      aria-label={t("Sidebar")}
      inert={!shown || undefined}
      className={cn(
        "bg-sidebar border-border flex h-full shrink-0 flex-col border-r transition-[width] duration-200 max-md:hidden",
        shown ? WORKSPACE_SIDEBAR_WIDTH_CLASS : "w-0 overflow-hidden border-r-0",
      )}
    >
      <WorkspaceSidebarBody />
    </aside>
  );
}
