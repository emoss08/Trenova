import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import { appModuleGroups } from "@/config/navigation.config";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { useMemo } from "react";
import { useSearchParams } from "react-router";
import { UpdateBanner } from "../update/update-banner";
import { FavoritesSection } from "./favorites-section";
import { NavUser } from "./nav-user";
import { OrganizationSwitcher } from "./organization-switcher";
import { QuickSearchTrigger } from "./quick-search-trigger";
import { SidebarNav } from "./sidebar-nav";

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const filteredModules = useFilteredNavigation();
  const [searchParams] = useSearchParams();
  const groupedModules = useMemo(() => {
    const modulesById = new Map(
      filteredModules.map((module) => [module.id, module]),
    );
    const groupedModuleIds = new Set<string>();
    const groups = appModuleGroups
      .map((group) => {
        const modules = group.moduleIds
          .map((moduleId) => {
            const module = modulesById.get(moduleId);
            if (module) {
              groupedModuleIds.add(moduleId);
            }
            return module;
          })
          .filter((module): module is (typeof filteredModules)[number] =>
            Boolean(module),
          );

        return {
          id: group.id,
          label: group.label,
          modules,
        };
      })
      .filter((group) => group.modules.length > 0);

    const ungroupedModules = filteredModules.filter(
      (module) => !groupedModuleIds.has(module.id),
    );

    if (ungroupedModules.length > 0) {
      groups.push({
        id: "navigation",
        label: "Navigation",
        modules: ungroupedModules,
      });
    }

    return groups;
  }, [filteredModules]);

  const hideAside = searchParams.get("hideAside") === "true";

  if (hideAside) {
    return null;
  }

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="h-14.5 items-center justify-center border-b border-border p-0 px-1">
        <OrganizationSwitcher />
      </SidebarHeader>
      <SidebarContent>
        <div className="px-2 pt-2">
          <QuickSearchTrigger />
        </div>
        <FavoritesSection />
        {groupedModules.map((group) => (
          <SidebarGroup key={group.id}>
            <SidebarGroupLabel className="font-semibold uppercase select-none">
              {group.label}
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarNav modules={group.modules} />
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <UpdateBanner />
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
