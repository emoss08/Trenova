import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigationStore } from "@/stores/navigation-store";
import { useUpdateStore } from "@/stores/update-store";
import { useEffect } from "react";
import { useLocation } from "react-router";
import { RouteCommandPalette } from "../command-palette/route-command-palette";
import { Header } from "../header";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";
import { PageHeader, type PageHeaderProps } from "../page-header";
import { CommandSidebar } from "./command-sidebar";

interface SidebarLayoutProps {
  children: React.ReactNode;
}

export function SidebarLayout({ children }: SidebarLayoutProps) {
  const location = useLocation();
  const filteredModules = useFilteredNavigation();
  const fetchStatus = useUpdateStore((state) => state.fetchStatus);
  const setActiveModuleId = useNavigationStore((state) => state.setActiveModuleId);
  const toggleSidebar = useNavigationStore((state) => state.toggleSidebar);

  useEffect(() => {
    void fetchStatus();
  }, [fetchStatus]);

  // Derive active module from current route
  useEffect(() => {
    const path = location.pathname;
    const matched = filteredModules.find((m) => {
      if (m.basePath === "/" && path === "/") return true;
      if (m.basePath !== "/" && m.basePath !== "#" && path.startsWith(m.basePath)) return true;
      if (m.id === "shipment" && path.startsWith("/shipment-management")) return true;
      if (m.id === "admin" && path.startsWith("/admin")) return true;
      return false;
    });
    if (matched) {
      setActiveModuleId(matched.id);
    }
  }, [location.pathname, filteredModules, setActiveModuleId]);

  // Keyboard shortcut: Ctrl+B toggles the sidebar
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

  return (
    <>
      <RouteCommandPalette />
      <KeyboardShortcutsDialog />
      <div className="flex h-screen overflow-hidden">
        <CommandSidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <Header />
          <main className="flex-1 overflow-y-auto">{children}</main>
        </div>
      </div>
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
