import { SidebarSectionLabel } from "@/components/navigation/sidebar-primitives";
import { buildCommandHref } from "@/components/command-palette/route-command-data";
import { navigationConfig } from "@/config/navigation.config";
import type { QuickActionCommand } from "@/config/navigation.types";
import { usePermissionStore } from "@/stores/permission-store";
import { Operation } from "@/types/permission";
import type { LucideIcon } from "lucide-react";
import { Building2Icon, MapPinIcon, TruckIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";

const SIDEBAR_QUICK_ACTIONS: Record<string, LucideIcon> = {
  "create-shipment": TruckIcon,
  "create-worker": UsersIcon,
  "create-location": MapPinIcon,
  "create-customer": Building2Icon,
};

interface SidebarQuickAction {
  definition: QuickActionCommand;
  icon: LucideIcon;
  href: string;
}

export function QuickActionsSection() {
  const hasPermission = usePermissionStore((state) => state.hasPermission);

  const actions = useMemo<SidebarQuickAction[]>(() => {
    const definitions = navigationConfig.quickActions ?? [];
    return definitions
      .filter((definition) => {
        const icon = SIDEBAR_QUICK_ACTIONS[definition.id];
        if (!icon) {
          return false;
        }
        if (!definition.resource) {
          return true;
        }
        return hasPermission(definition.resource, definition.requiredOperation ?? Operation.Create);
      })
      .map((definition) => ({
        definition,
        icon: SIDEBAR_QUICK_ACTIONS[definition.id],
        href: buildCommandHref(definition.path, definition.query),
      }));
  }, [hasPermission]);

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
