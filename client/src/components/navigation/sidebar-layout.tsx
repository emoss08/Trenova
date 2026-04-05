import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { cn } from "@/lib/utils";
import { useNavigationStore } from "@/stores/navigation-store";
import { useUpdateStore } from "@/stores/update-store";
import type { ModuleId } from "@/config/navigation.types";
import { getFirstNavPath } from "@/config/navigation.types";
import { useCallback, useEffect, useMemo } from "react";
import { useLocation, useNavigate } from "react-router";
import { RouteCommandPalette } from "../command-palette/route-command-palette";
import { Header } from "../header";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";
import { PageHeader, type PageHeaderProps } from "../page-header";
import { FavoritesPanel } from "./favorites-panel";
import { IconRail } from "./icon-rail";
import { ModulePanel } from "./module-panel";

interface SidebarLayoutProps {
  children: React.ReactNode;
}

export function SidebarLayout({ children }: SidebarLayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const filteredModules = useFilteredNavigation();
  const fetchStatus = useUpdateStore((state) => state.fetchStatus);
  const {
    activeModuleId,
    setActiveModuleId,
    modulePanelCollapsed,
    setModulePanelCollapsed,
    toggleModulePanel,
    panelView,
    setPanelView,
  } = useNavigationStore();

  useEffect(() => {
    void fetchStatus();
  }, [fetchStatus]);

  // Derive active module from current route
  useEffect(() => {
    const path = location.pathname;
    const matched = filteredModules.find((m) => {
      if (m.basePath === "/" && path === "/") return true;
      if (m.basePath !== "/" && path.startsWith(m.basePath)) return true;
      // Check admin routes
      if (m.id === "admin" && path.startsWith("/admin")) return true;
      return false;
    });
    if (matched) {
      setActiveModuleId(matched.id);
      setPanelView("module");
    }
  }, [location.pathname, filteredModules, setActiveModuleId, setPanelView]);

  const activeModule = useMemo(
    () => filteredModules.find((m) => m.id === activeModuleId) ?? null,
    [filteredModules, activeModuleId],
  );

  const handleModuleSelect = useCallback(
    (id: ModuleId) => {
      setPanelView("module");
      if (id === activeModuleId && panelView === "module") {
        toggleModulePanel();
        return;
      }
      setActiveModuleId(id);
      if (modulePanelCollapsed) {
        setModulePanelCollapsed(false);
      }
      const mod = filteredModules.find((m) => m.id === id);
      if (mod) {
        const targetPath = getFirstNavPath(mod);
        void navigate(targetPath);
      }
    },
    [activeModuleId, filteredModules, navigate, setActiveModuleId, toggleModulePanel, setPanelView, panelView, modulePanelCollapsed, setModulePanelCollapsed],
  );

  const handleFavoritesSelect = useCallback(() => {
    if (panelView === "favorites") {
      toggleModulePanel();
    } else {
      setPanelView("favorites");
      if (modulePanelCollapsed) {
        setModulePanelCollapsed(false);
      }
    }
  }, [panelView, toggleModulePanel, setPanelView, modulePanelCollapsed, setModulePanelCollapsed]);

  // Keyboard shortcut: Ctrl+B toggles module panel
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "b") {
        e.preventDefault();
        toggleModulePanel();
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [toggleModulePanel]);

  return (
    <>
      <RouteCommandPalette />
      <KeyboardShortcutsDialog />
      <div className="flex h-screen overflow-hidden">
        <IconRail
          modules={filteredModules}
          activeModuleId={panelView === "favorites" ? null : activeModuleId}
          onModuleSelect={handleModuleSelect}
          isFavoritesActive={panelView === "favorites" && !modulePanelCollapsed}
          onFavoritesSelect={handleFavoritesSelect}
        />
        {panelView === "favorites" ? (
          <FavoritesPanel
            collapsed={modulePanelCollapsed}
            onToggleCollapse={toggleModulePanel}
          />
        ) : (
          activeModule && (!activeModule.hideSecondarySidebar || activeModule.id === "admin") && (
            <ModulePanel
              module={activeModule}
              collapsed={modulePanelCollapsed}
              onToggleCollapse={toggleModulePanel}
            />
          )
        )}
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
