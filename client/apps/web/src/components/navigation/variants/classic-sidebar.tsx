import { ActivitySection } from "@/components/navigation/activity-section";
import { AttentionSection } from "@/components/navigation/attention-section";
import { BrowseSection } from "@/components/navigation/browse-section";
import { CustomizeSidebarDialog } from "@/components/navigation/customize-sidebar-dialog";
import { FavoritesSection } from "@/components/navigation/favorites-section";
import { OrgSwitcher } from "@/components/navigation/org-switcher";
import { QuickActionsSection } from "@/components/navigation/quick-actions-section";
import { SearchTrigger } from "@/components/navigation/sidebar-chrome";
import { UserMenu } from "@/components/navigation/user-menu";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { type SidebarSectionKey } from "@/lib/graphql/sidebar-preferences";
import { cn } from "@trenova/shared/lib/utils";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useNavigationStore } from "@/stores/navigation-store";
import type { ComponentType } from "react";

const SECTION_COMPONENTS: Record<SidebarSectionKey, ComponentType> = {
  attention: AttentionSection,
  quickActions: QuickActionsSection,
  favorites: FavoritesSection,
  activity: ActivitySection,
  browse: BrowseSection,
};

/**
 * The sidebar as it shipped before the layout review: every section stacked
 * in the order the person arranged them.
 */
export function ClassicSidebar() {
  const collapsed = useNavigationStore((state) => state.sidebarCollapsed);
  const { data: preferences } = useSidebarPreferences();

  const sections = (preferences?.sections ?? [])
    .filter((section) => !section.hidden)
    .map((section) => ({
      key: section.key,
      Section: SECTION_COMPONENTS[section.key as SidebarSectionKey],
    }))
    .filter((entry) => entry.Section != null);

  return (
    <aside
      className={cn(
        "border-border bg-sidebar flex h-screen flex-col border-r transition-[width] duration-200",
        collapsed ? "w-0 overflow-hidden border-r-0" : "w-64",
      )}
    >
      <div className="flex w-64 flex-col gap-2 px-2 pt-2 pb-1">
        <OrgSwitcher />
        <div className="flex items-center gap-1.5">
          <SearchTrigger />
          <CustomizeSidebarDialog />
        </div>
      </div>

      <ScrollArea className="w-64 flex-1" maskHeight={20}>
        <nav className="flex flex-col gap-4 px-2 pt-1 pb-4">
          {sections.map(({ key, Section }) => (
            <Section key={key} />
          ))}
        </nav>
      </ScrollArea>

      <div className="border-border w-64 border-t p-1.5">
        <UserMenu />
      </div>
    </aside>
  );
}
