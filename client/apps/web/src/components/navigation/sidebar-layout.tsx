import { hotkeyOf } from "@/config/keybinds.config";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { getPageTitle } from "@/lib/route-utils";
import { usePageAccessOffered } from "@/hooks/use-permission";
import { lazyComponent } from "@/lib/lazy-component";
import { useAppDialogPrepared, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useNavigationStore } from "@/stores/navigation-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { recordAtLocation } from "@/config/record-links";
import { isGlobalSearchEntityType } from "@/services/global-search";
import { useRecentPagesStore } from "@/stores/recent-pages-store";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import { useUpdateStore } from "@/stores/update-store";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Suspense, useEffect } from "react";
import { useLocation } from "react-router";
import { AssistantWidget, useAssistantSideInset } from "../assistant/assistant-widget";
import { CommandPaletteMount } from "../command-palette/command-palette-mount";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";
import { PageHeader, type PageHeaderProps } from "../page-header";
import {
  ShellFrameContext,
  usePageBarSlots,
  useRouteFrame,
  useRouteRoomRail,
  useShellFrame,
} from "./shell-frame";
import { useSidebarFold } from "./sidebar-fold";
import { findModuleForPath } from "./sidebar-model";
import { useSettingsShortcut } from "./use-settings-shortcut";
import { RailShell } from "./rail-shell";

// The page access panel is read once it is asked for (its user menu entry
// pointed at, or its keybind); the shell holds nothing of it but this import.
const PageAccessSheet = lazyComponent(
  () => import("../page-access/page-access-sheet"),
  (module) => module.PageAccessSheet,
);

const togglePageAccess = () => useAppDialogsStore.getState().toggleDialog("page-access");

interface SidebarLayoutProps {
  children: React.ReactNode;
}

/**
 * Remembers where the person has been so the modules menu can offer the way
 * back. The title follows the breadcrumb once a record page has loaded its
 * name, so a detail page is remembered by what it is, not by "Details".
 */
function useRecordRecentPages() {
  const { pathname } = useLocation();
  const breadcrumbs = useBreadcrumbs();
  const recordVisit = useRecentPagesStore((state) => state.recordVisit);
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const lastCrumb = breadcrumbs.at(-1)?.crumb;
  const title = typeof lastCrumb === "string" ? lastCrumb : getPageTitle(pathname);

  useEffect(() => {
    recordVisit(organizationId, { path: pathname, title });
  }, [organizationId, pathname, title, recordVisit]);
}

/**
 * Brings a record the palette already remembers back to the front when it is
 * opened some other way: from a table row, a link in a notification, a
 * mention in a reply.
 */
function useTouchRecentRecords() {
  const { pathname, search } = useLocation();
  const touch = useRecentRecordsStore((state) => state.touch);
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);

  useEffect(() => {
    const record = recordAtLocation(pathname, search);
    if (!record || record.entityId === "" || !isGlobalSearchEntityType(record.entityType)) {
      return;
    }
    touch(organizationId, record.entityType, record.entityId);
  }, [organizationId, pathname, search, touch]);
}

/**
 * `railStands` is a room whose own rail never folds: beside the page ⌘B has
 * nothing to fold there, so it leaves the kept fold alone; a phone's sheet
 * still answers it.
 */
function useLayoutEffects(railStands: boolean) {
  const location = useLocation();
  const filteredModules = useFilteredNavigation();
  const fetchStatus = useUpdateStore((state) => state.fetchStatus);
  const setActiveModuleId = useNavigationStore((state) => state.setActiveModuleId);
  const setCompactSidebarOpen = useNavigationStore((state) => state.setCompactSidebarOpen);
  const { compact, toggle: toggleSidebar } = useSidebarFold();

  useEffect(() => {
    void fetchStatus();
  }, [fetchStatus]);

  useEffect(() => {
    const matched = findModuleForPath(filteredModules, location.pathname);
    if (matched) {
      setActiveModuleId(matched.id);
    }
  }, [location.pathname, filteredModules, setActiveModuleId]);

  useEffect(() => {
    setCompactSidebarOpen(false);
  }, [location.pathname, setCompactSidebarOpen]);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "b") {
        e.preventDefault();
        if (compact || !railStands) {
          toggleSidebar();
        }
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [toggleSidebar, compact, railStands]);

  useRecordRecentPages();
  useTouchRecentRecords();
  useSettingsShortcut();

  useHotkey(hotkeyOf("general", "page-access"), togglePageAccess, {
    enabled: usePageAccessOffered(),
  });
}

export function SidebarLayout({ children }: SidebarLayoutProps) {
  const assistantSide = useAssistantSideInset();
  const frame = useRouteFrame();
  const roomRail = useRouteRoomRail();
  const railStands = frame === "room" && roomRail === "stands";
  useLayoutEffects(railStands);
  const pageAccessPrepared = useAppDialogPrepared("page-access");

  return (
    <>
      <CommandPaletteMount />
      <KeyboardShortcutsDialog />
      {pageAccessPrepared ? (
        <Suspense fallback={null}>
          <PageAccessSheet.Component />
        </Suspense>
      ) : null}
      <AssistantWidget />
      {/* With the assistant docked down one side, the page moves over for it
          rather than sitting under it. */}
      <div className="as-reflow" data-assistant-side={assistantSide ?? undefined}>
        <ShellFrameContext.Provider value={frame}>
          <RailShell room={frame === "room"} railStands={railStands}>
            {children}
          </RailShell>
        </ShellFrameContext.Provider>
      </div>
    </>
  );
}

type PageLayoutProps = {
  pageHeaderProps: PageHeaderProps;
  children: React.ReactNode;
  className?: string;
  fill?: boolean;
  /**
   * Lays the body edge to edge for a workspace that is more than a table but
   * still bleeds one, such as a board with a header block above its table.
   * The table's chrome answers through the same `bleed:` variant.
   */
  bleed?: boolean;
};

export function PageLayout({
  pageHeaderProps,
  children,
  className,
  fill = false,
  bleed = false,
}: PageLayoutProps) {
  const frame = useShellFrame();
  const shellSlots = usePageBarSlots();
  const slots = frame === "page" ? shellSlots : null;

  return (
    <div
      data-slot="page"
      className={cn(
        "flex min-h-full flex-col has-[>[data-slot=page-body]>[data-slot=data-table]:only-child]:h-full",
        fill && "h-full",
      )}
    >
      <PageHeader
        {...pageHeaderProps}
        visuallyHidden={frame === "room"}
        leading={slots?.leading(pageHeaderProps.title)}
        trailing={slots?.trailing}
      />
      <div
        data-slot="page-body"
        data-bleed={bleed || undefined}
        className={cn(
          "flex min-w-0 flex-1 flex-col gap-y-4 p-4",
          "has-[>[data-slot=data-table]:only-child]:min-h-0 has-[>[data-slot=data-table]:only-child]:gap-y-0 has-[>[data-slot=data-table]:only-child]:p-0",
          fill && "min-h-0 overflow-y-auto",
          bleed && "min-h-0 gap-y-0 p-0",
          className,
        )}
      >
        {children}
      </div>
    </div>
  );
}
