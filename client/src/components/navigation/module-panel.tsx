import { useMemo, useState } from "react";
import { Link, useLocation } from "react-router";
import { ChevronLeftIcon, ChevronRightIcon, ChevronsUpDownIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { BetaTag } from "@/components/beta-tag";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { adminLinks } from "@/config/navigation.config";
import type { NavModule, NavItem, NavGroup } from "@/config/navigation.types";
import { isNavGroup } from "@/config/navigation.types";
import { usePermissionStore } from "@/stores/permission-store";
import type { SidebarLink } from "@/components/sidebar-nav";

type ModulePanelProps = {
  module: NavModule;
  collapsed: boolean;
  onToggleCollapse: () => void;
};

function isRouteActive(currentPath: string, itemPath?: string): boolean {
  if (!itemPath) return false;
  if (itemPath === "/") return currentPath === "/";
  const a = currentPath.endsWith("/") ? currentPath.slice(0, -1) : currentPath;
  const b = itemPath.endsWith("/") ? itemPath.slice(0, -1) : itemPath;
  return a.startsWith(b);
}

function NavItemLink({ item, currentPath }: { item: NavItem; currentPath: string }) {
  const active = isRouteActive(currentPath, item.path);

  return (
    <Link
      to={item.path}
      className={cn(
        "flex items-center gap-2 rounded-md px-2 h-6 text-base transition-colors",
        active
          ? "bg-accent text-accent-foreground font-medium"
          : "text-foreground/70 hover:bg-muted hover:text-foreground",
        item.disabled && "opacity-40 pointer-events-none",
      )}
      aria-disabled={item.disabled}
      tabIndex={item.disabled ? -1 : undefined}
    >
      <span className="truncate">{item.label}</span>
      {item.includeBetaTag && <BetaTag className="ml-auto" />}
    </Link>
  );
}

function NavGroupSection({
  group,
  currentPath,
}: {
  group: NavGroup;
  currentPath: string;
}) {
  const hasActiveChild = group.items.some((item) => isRouteActive(currentPath, item.path));
  const [open, setOpen] = useState(hasActiveChild || group.defaultOpen || false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        render={(props) => (
          <button
            {...props}
            className={cn(
              "flex w-full items-center justify-between rounded-md px-2 h-6 text-base",
              "text-foreground/70 hover:bg-muted hover:text-foreground transition-colors",
              hasActiveChild && "text-foreground font-medium",
            )}
          >
            <span className="truncate">{group.label}</span>
            <ChevronsUpDownIcon
              className={cn("size-3 shrink-0 transition-transform", open && "rotate-180")}
            />
          </button>
        )}
      />
      <CollapsibleContent>
        <div className="ml-2 border-l border-border pl-2 mt-0.5 flex flex-col gap-0.5">
          {group.items.map((item) => (
            <NavItemLink key={item.id} item={item} currentPath={currentPath} />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function AdminSection({ currentPath }: { currentPath: string }) {
  const manifest = usePermissionStore((s) => s.manifest);
  const isPlatformAdmin = manifest?.isPlatformAdmin ?? false;

  const grouped = useMemo(() => {
    const filtered = adminLinks.filter(
      (link) => !link.platformAdminOnly || isPlatformAdmin,
    );

    const groups = new Map<string, SidebarLink[]>();
    for (const link of filtered) {
      const groupName = link.group ?? "Other";
      const existing = groups.get(groupName);
      if (existing) {
        existing.push(link);
      } else {
        groups.set(groupName, [link]);
      }
    }
    return groups;
  }, [isPlatformAdmin]);

  return (
    <div className="flex flex-col gap-4">
      {Array.from(grouped.entries()).map(([groupName, links]) => (
        <div key={groupName} className="flex flex-col gap-0.5">
          <span className="px-2 text-xs font-semibold uppercase tracking-wide text-foreground/50 select-none mb-1">
            {groupName}
          </span>
          {links.map((link) => {
            const active = isRouteActive(currentPath, link.href);
            return (
              <Link
                key={`${groupName}:${link.title}:${link.href}`}
                to={link.href}
                className={cn(
                  "flex items-center gap-2 rounded-md px-2 h-6 text-base transition-colors",
                  active
                    ? "bg-accent text-accent-foreground font-medium"
                    : "text-foreground/70 hover:bg-muted hover:text-foreground",
                  link.disabled && "opacity-40 pointer-events-none",
                )}
                aria-disabled={link.disabled}
                tabIndex={link.disabled ? -1 : undefined}
              >
                <span className="truncate">{link.title}</span>
                {link.includeBetaTag && <BetaTag className="ml-auto" />}
              </Link>
            );
          })}
        </div>
      ))}
    </div>
  );
}

export function ModulePanel({ module, collapsed, onToggleCollapse }: ModulePanelProps) {
  const { pathname } = useLocation();

  if (collapsed) {
    return (
      <div className="w-0 overflow-hidden transition-all duration-200" />
    );
  }

  const isAdmin = module.id === "admin";

  return (
    <div className="flex h-full w-[200px] flex-col border-r border-border bg-background transition-all duration-200">
      <div className="flex items-center justify-between px-3 py-3">
        <h2 className="text-base font-semibold truncate">{module.label}</h2>
        <button
          onClick={onToggleCollapse}
          className="flex size-5 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
        >
          {collapsed ? (
            <ChevronRightIcon className="size-3.5" />
          ) : (
            <ChevronLeftIcon className="size-3.5" />
          )}
        </button>
      </div>

      <ScrollArea className="flex-1" maskHeight={20}>
        <div className="flex flex-col gap-0.5 px-2 pb-3">
          {isAdmin ? (
            <AdminSection currentPath={pathname} />
          ) : (
            module.navigation.map((item) =>
              isNavGroup(item) ? (
                <NavGroupSection key={item.id} group={item} currentPath={pathname} />
              ) : (
                <NavItemLink key={item.id} item={item} currentPath={pathname} />
              ),
            )
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
