import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { getPageTitle } from "@/lib/route-utils";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigationStore } from "@/stores/navigation-store";
import { useRecentPagesStore } from "@/stores/recent-pages-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useUpdateStore } from "@/stores/update-store";
import { useEffect } from "react";
import { useLocation } from "react-router";
import { CommandPaletteMount } from "../command-palette/command-palette-mount";
import { Header } from "../header";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";
import { PageHeader, type PageHeaderProps } from "../page-header";
import { findModuleForPath } from "./sidebar-model";
import { ClassicSidebar } from "./variants/classic-sidebar";
import { WorkspaceContextBar, WorkspaceHeader } from "./workspace-header";
import { WorkspaceSidebar } from "./workspace-sidebar";

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

function useLayoutEffects() {
  const location = useLocation();
  const filteredModules = useFilteredNavigation();
  const fetchStatus = useUpdateStore((state) => state.fetchStatus);
  const setActiveModuleId = useNavigationStore((state) => state.setActiveModuleId);
  const toggleSidebar = useNavigationStore((state) => state.toggleSidebar);

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
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "b") {
        e.preventDefault();
        toggleSidebar();
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [toggleSidebar]);

  useRecordRecentPages();
}

function WorkspaceShell({ children }: SidebarLayoutProps) {
  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <WorkspaceHeader />
      <div className="flex min-h-0 flex-1">
        <WorkspaceSidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <WorkspaceContextBar />
          <main className="min-w-0 flex-1 overflow-y-auto">{children}</main>
        </div>
      </div>
    </div>
  );
}

function ClassicShell({ children }: SidebarLayoutProps) {
  return (
    <div className="flex h-screen overflow-hidden">
      <ClassicSidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
    </div>
  );
}

export function SidebarLayout({ children }: SidebarLayoutProps) {
  const variant = useNavigationStore((state) => state.sidebarVariant);
  useLayoutEffects();

  const Shell = variant === "classic" ? ClassicShell : WorkspaceShell;

  return (
    <>
      <CommandPaletteMount />
      <KeyboardShortcutsDialog />
      <Shell>{children}</Shell>
    </>
  );
}

export function PageLayout({
  pageHeaderProps,
  children,
  className,
}: {
  pageHeaderProps: PageHeaderProps;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <>
      <PageHeader {...pageHeaderProps} />
      <div className={cn("flex flex-col gap-y-4 p-4", className)}>{children}</div>
    </>
  );
}

export function AdminPageLayout({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("flex flex-col", className)}>{children}</div>;
}
