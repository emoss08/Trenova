import { ActivitySection } from "@/components/navigation/activity-section";
import { AttentionSection } from "@/components/navigation/attention-section";
import { BrowseSection } from "@/components/navigation/browse-section";
import { CustomizeSidebarDialog } from "@/components/navigation/customize-sidebar-dialog";
import { FavoritesSection } from "@/components/navigation/favorites-section";
import { OrgSwitcher } from "@/components/navigation/org-switcher";
import { QuickActionsSection } from "@/components/navigation/quick-actions-section";
import { UserMenu } from "@/components/navigation/user-menu";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { type SidebarSectionKey } from "@/lib/graphql/sidebar-preferences";
import { cn } from "@trenova/shared/lib/utils";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useNavigationStore } from "@/stores/navigation-store";
import { Search } from "lucide-react";
import type { ComponentType } from "react";

const SECTION_COMPONENTS: Record<SidebarSectionKey, ComponentType> = {
  attention: AttentionSection,
  quickActions: QuickActionsSection,
  favorites: FavoritesSection,
  activity: ActivitySection,
  browse: BrowseSection,
};

function SearchTrigger() {
  const setOpen = useCommandPaletteStore((state) => state.setOpen);

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="flex h-7 w-full items-center gap-2 rounded-md border border-border bg-background px-2 text-xs text-muted-foreground transition-colors hover:border-ring/40 hover:text-foreground"
    >
      <Search className="size-3.5 shrink-0" strokeWidth={1.75} />
      <span className="flex-1 truncate text-left">Search or jump to…</span>
      <Kbd>⌘K</Kbd>
    </button>
  );
}

export function CommandSidebar() {
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
        "flex h-screen flex-col border-r border-border bg-sidebar transition-[width] duration-200",
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

      <div className="w-64 border-t border-border p-1.5">
        <UserMenu />
      </div>
    </aside>
  );
}
