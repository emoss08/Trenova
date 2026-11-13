import type { SidebarModuleView, SidebarPageItem } from "@/components/navigation/sidebar-model";

export interface SectionIndexPanel {
  id: string;
  /** The group's name, or null for pages the module lists outside any group. */
  label: string | null;
  /** The module's configuration catalogue, which the sidebar sets apart under its own name. */
  configuration: boolean;
  items: SidebarPageItem[];
}

const CONFIGURATION_PANEL_ID = "configuration";

/**
 * The panels a section index draws from the module's view as the sidebar has
 * it, so it lists exactly the pages the sidebar offers the person and in the
 * same order. A module section takes every group and the configuration
 * catalogue last; a group section takes that group's pages alone. A disabled
 * entry is not a page anyone can open, and a panel left with nothing is
 * dropped.
 */
export function sectionIndexPanels(
  view: SidebarModuleView,
  groupId: string | null,
): SectionIndexPanel[] {
  const openable = (items: readonly SidebarPageItem[]) => items.filter((item) => !item.disabled);

  if (groupId !== null) {
    const items = openable(
      [...view.sections.flatMap((section) => section.items), ...view.configuration].filter(
        (item) => item.groupId === groupId,
      ),
    );
    return items.length > 0 ? [{ id: groupId, label: null, configuration: false, items }] : [];
  }

  const panels: SectionIndexPanel[] = view.sections.map((section) => ({
    id: section.id,
    label: section.label,
    configuration: false,
    items: openable(section.items),
  }));
  panels.push({
    id: CONFIGURATION_PANEL_ID,
    label: null,
    configuration: true,
    items: openable(view.configuration),
  });

  return panels.filter((panel) => panel.items.length > 0);
}
