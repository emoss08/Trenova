import {
  buildModuleView,
  collectNavPaths,
  findModuleForPath,
  groupModulesByDomain,
  moduleAttention,
  pageAttention,
  type ModuleAttention,
  type SidebarDomain,
  type SidebarModuleView,
} from "@/components/navigation/sidebar-model";
import type { ModuleId, NavModule } from "@/config/navigation.types";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { useAttentionSummary } from "@/hooks/use-attention";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { SIDEBAR_SECTION_KEYS, type SidebarSectionKey } from "@/lib/graphql/sidebar-preferences";
import { findActiveNavPath } from "@/lib/route-utils";
import { useMemo } from "react";
import { useLocation } from "react-router";

export interface SidebarNavigation {
  /** The home module, when the person may see it. */
  home: NavModule | null;
  /** The settings module, present only when at least one admin page is reachable. */
  settings: NavModule | null;
  /** Every working module the person may see, in their configured domains. */
  domains: SidebarDomain[];
  views: ReadonlyMap<ModuleId, SidebarModuleView>;
  activeModule: NavModule | null;
  /** The single nav path that best matches the location, or null on a page no entry owns. */
  activePath: string | null;
  attention: ReadonlyMap<ModuleId, ModuleAttention>;
  /** The same counts keyed by the page that owns them. */
  pageAttention: ReadonlyMap<string, ModuleAttention>;
  /** Sections the person hid in the sidebar preferences. */
  hiddenSections: ReadonlySet<SidebarSectionKey>;
}

/**
 * The one place every sidebar layout reads its data from. Layouts differ in
 * how they draw modules, not in which modules exist, which page is active or
 * how much work is waiting, so those answers are computed once here.
 */
export function useSidebarNavigation(): SidebarNavigation {
  const { pathname } = useLocation();
  const modules = useFilteredNavigation();
  const adminLinks = useAccessibleAdminLinks();
  const { data: summary } = useAttentionSummary();
  const { data: preferences } = useSidebarPreferences();

  const hiddenSections = useMemo(() => {
    const hidden = new Set<SidebarSectionKey>();
    for (const section of preferences?.sections ?? []) {
      if (section.hidden) {
        hidden.add(section.key as SidebarSectionKey);
      }
    }
    return hidden;
  }, [preferences?.sections]);

  const visibleModules = useMemo(
    () => modules.filter((module) => module.id !== "admin" || adminLinks.length > 0),
    [modules, adminLinks],
  );

  const views = useMemo(
    () =>
      new Map<ModuleId, SidebarModuleView>(
        visibleModules.map((module) => [module.id, buildModuleView(module, adminLinks)]),
      ),
    [visibleModules, adminLinks],
  );

  const home = useMemo(
    () => visibleModules.find((module) => module.id === "home") ?? null,
    [visibleModules],
  );
  const settings = useMemo(
    () => visibleModules.find((module) => module.id === "admin") ?? null,
    [visibleModules],
  );

  const domains = useMemo(
    () =>
      groupModulesByDomain(
        visibleModules.filter((module) => module.id !== "home" && module.id !== "admin"),
      ),
    [visibleModules],
  );

  const activeModule = useMemo(
    () => findModuleForPath(visibleModules, pathname),
    [visibleModules, pathname],
  );

  const candidatePaths = useMemo(() => collectNavPaths(Array.from(views.values())), [views]);
  const activePath = useMemo(
    () => findActiveNavPath(pathname, candidatePaths),
    [pathname, candidatePaths],
  );

  const attentionMetrics = preferences?.attentionMetrics;
  const attentionHidden = hiddenSections.has(SIDEBAR_SECTION_KEYS.attention);
  const attention = useMemo(() => {
    if (attentionHidden) {
      return new Map<ModuleId, ModuleAttention>();
    }
    return moduleAttention(visibleModules, summary, attentionMetrics ?? []);
  }, [visibleModules, summary, attentionMetrics, attentionHidden]);
  const byPage = useMemo(() => {
    if (attentionHidden) {
      return new Map<string, ModuleAttention>();
    }
    return pageAttention(summary, attentionMetrics ?? []);
  }, [summary, attentionMetrics, attentionHidden]);

  return {
    home,
    settings,
    domains,
    views,
    activeModule,
    activePath,
    attention,
    pageAttention: byPage,
    hiddenSections,
  };
}
