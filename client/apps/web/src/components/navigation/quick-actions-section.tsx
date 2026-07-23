import { buildCommandHref } from "@/components/command-palette/route-command-data";
import { SidebarSectionLabel } from "@/components/navigation/sidebar-primitives";
import { navigationConfig } from "@/config/navigation.config";
import type { QuickActionCommand } from "@/config/navigation.types";
import { QUICK_ACTION_ICONS } from "@/config/quick-action-icons";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";
import type { LucideIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";

interface SidebarQuickAction {
  definition: QuickActionCommand;
  icon: LucideIcon;
  href: string;
}

export function QuickActionsSection() {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const { data: preferences } = useSidebarPreferences();
  const quickActionIds = preferences?.quickActionIds;

  const actions = useMemo<SidebarQuickAction[]>(() => {
    const definitions = navigationConfig.quickActions ?? [];
    const definitionsById = new Map(definitions.map((definition) => [definition.id, definition]));

    return (quickActionIds ?? [])
      .map((id) => definitionsById.get(id))
      .filter((definition): definition is QuickActionCommand => {
        if (!definition || !QUICK_ACTION_ICONS[definition.id]) {
          return false;
        }
        if (!definition.resource) {
          return true;
        }
        return hasPermission(definition.resource, definition.requiredOperation ?? Operation.Create);
      })
      .map((definition) => ({
        definition,
        icon: QUICK_ACTION_ICONS[definition.id],
        href: buildCommandHref(definition.path, definition.query),
      }));
  }, [hasPermission, quickActionIds]);

  if (actions.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1">
      <SidebarSectionLabel>Quick Actions</SidebarSectionLabel>
      <div className="grid grid-cols-2 gap-1.5 px-0.5">
        {actions.map(({ definition, icon: Icon, href }) => (
          <Link
            key={definition.id}
            to={href}
            title={definition.description}
            className="flex h-7 items-center gap-1.5 truncate rounded-md border border-border bg-background px-2 text-xs font-medium text-foreground/80 transition-colors hover:bg-muted hover:text-foreground"
          >
            <Icon className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} />
            <span className="truncate">{definition.label.replace(/^Create /, "New ")}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
