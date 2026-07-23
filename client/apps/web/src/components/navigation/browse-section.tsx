import { BetaTag } from "@/components/beta-tag";
import { NavItemBadge } from "@/components/navigation/nav-item-badge";
import { SidebarNavLink, SidebarSectionLabel } from "@/components/navigation/sidebar-primitives";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import type { SidebarLink } from "@/components/sidebar-nav";
import type { NavGroup, NavItem, NavModule } from "@/config/navigation.types";
import { isNavGroup } from "@/config/navigation.types";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { findActiveNavPath, isRouteActive } from "@/lib/route-utils";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigationStore } from "@/stores/navigation-store";
import { ChevronRightIcon, ChevronsUpDownIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useLocation } from "react-router";

function NavItemRow({ item, activePath }: { item: NavItem; activePath: string | null }) {
  return (
    <SidebarNavLink
      to={item.path}
      active={item.path === activePath}
      disabled={item.disabled}
    >
      <span className="truncate">{item.label}</span>
      {item.badge ? (
        <span className="ml-auto flex items-center gap-1">
          <NavItemBadge badge={item.badge} />
          {item.includeBetaTag && <BetaTag />}
        </span>
      ) : (
        item.includeBetaTag && <BetaTag className="ml-auto" />
      )}
    </SidebarNavLink>
  );
}

function NavGroupSection({ group, activePath }: { group: NavGroup; activePath: string | null }) {
  const hasActiveChild = group.items.some((item) => item.path === activePath);
  const [open, setOpen] = useState(hasActiveChild || group.defaultOpen || false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        render={(props) => (
          <button
            {...props}
            className={cn(
              "flex h-6 w-full items-center justify-between rounded-md px-2 text-base",
              "text-foreground/70 transition-colors hover:bg-muted hover:text-foreground",
              hasActiveChild && "font-medium text-foreground",
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
        <div className="mt-0.5 ml-2 flex flex-col gap-0.5 border-l border-border pl-2">
          {group.items.map((item) => (
            <NavItemRow key={item.id} item={item} activePath={activePath} />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function AdminLinkGroups({ links, activePath }: { links: SidebarLink[]; activePath: string | null }) {
  const grouped = useMemo(() => {
    const groups = new Map<string, SidebarLink[]>();
    for (const link of links) {
      const groupName = link.group ?? "Other";
      const existing = groups.get(groupName);
      if (existing) {
        existing.push(link);
      } else {
        groups.set(groupName, [link]);
      }
    }
    return groups;
  }, [links]);

  return (
    <div className="flex flex-col gap-2">
      {Array.from(grouped.entries()).map(([groupName, groupLinks]) => (
        <div key={groupName} className="flex flex-col gap-0.5">
          <span className="px-2 pt-1 text-2xs font-semibold tracking-wide text-foreground/50 uppercase select-none">
            {groupName}
          </span>
          {groupLinks.map((link) => (
            <SidebarNavLink
              key={`${groupName}:${link.title}:${link.href}`}
              to={link.href}
              active={link.href === activePath}
              disabled={link.disabled}
            >
              <span className="truncate">{link.title}</span>
              {link.includeBetaTag && <BetaTag className="ml-auto" />}
            </SidebarNavLink>
          ))}
        </div>
      ))}
    </div>
  );
}

function ModuleSection({
  module,
  currentPath,
  open,
  onOpenChange,
  children,
}: {
  module: NavModule;
  currentPath: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}) {
  const Icon = module.icon;
  const isActive = currentPath !== "/" && isRouteActive(currentPath, module.basePath);

  return (
    <Collapsible open={open} onOpenChange={onOpenChange}>
      <CollapsibleTrigger
        render={(props) => (
          <button
            {...props}
            className={cn(
              "flex h-7 w-full items-center gap-2 rounded-md px-2 text-base transition-colors",
              "text-foreground/80 hover:bg-muted hover:text-foreground",
              (isActive || open) && "font-medium text-foreground",
            )}
          >
            <Icon className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.75} />
            <span className="min-w-0 flex-1 truncate text-left">{module.label}</span>
            <ChevronRightIcon
              className={cn(
                "size-3.5 shrink-0 text-muted-foreground transition-transform",
                open && "rotate-90",
              )}
            />
          </button>
        )}
      />
      <CollapsibleContent>
        <div className="mt-0.5 ml-3 flex flex-col gap-0.5 border-l border-border py-0.5 pl-2">
          {children}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

export function BrowseSection() {
  const { pathname } = useLocation();
  const modules = useFilteredNavigation();
  const adminLinks = useAccessibleAdminLinks();
  const activeModuleId = useNavigationStore((state) => state.activeModuleId);
  const [moduleOverrides, setModuleOverrides] = useState<ReadonlyMap<string, boolean>>(
    () => new Map(),
  );

  const activePath = useMemo(() => {
    const candidatePaths: string[] = [];
    for (const module of modules) {
      for (const item of module.navigation) {
        if (isNavGroup(item)) {
          for (const child of item.items) {
            candidatePaths.push(child.path);
          }
        } else {
          candidatePaths.push(item.path);
        }
      }
    }
    for (const link of adminLinks) {
      candidatePaths.push(link.href);
    }
    return findActiveNavPath(pathname, candidatePaths);
  }, [modules, adminLinks, pathname]);

  const isModuleOpen = (moduleId: string) =>
    moduleOverrides.get(moduleId) ?? moduleId === activeModuleId;

  const setModuleOpen = (moduleId: string, open: boolean) => {
    setModuleOverrides((previous) => {
      const next = new Map(previous);
      next.set(moduleId, open);
      return next;
    });
  };

  return (
    <div className="flex flex-col gap-0.5">
      <SidebarSectionLabel>Browse</SidebarSectionLabel>
      {modules.map((module) => {
        if (module.id === "home") {
          const Icon = module.icon;
          return (
            <SidebarNavLink key={module.id} to={module.basePath} active={pathname === "/"} className="h-7">
              <Icon className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.75} />
              <span className="truncate">{module.label}</span>
            </SidebarNavLink>
          );
        }

        if (module.id === "admin") {
          if (adminLinks.length === 0) {
            return null;
          }
          return (
            <ModuleSection
              key={module.id}
              module={module}
              currentPath={pathname}
              open={isModuleOpen(module.id)}
              onOpenChange={(open) => setModuleOpen(module.id, open)}
            >
              <AdminLinkGroups links={adminLinks} activePath={activePath} />
            </ModuleSection>
          );
        }

        return (
          <ModuleSection
            key={module.id}
            module={module}
            currentPath={pathname}
            open={isModuleOpen(module.id)}
            onOpenChange={(open) => setModuleOpen(module.id, open)}
          >
            {module.navigation.map((item) =>
              isNavGroup(item) ? (
                <NavGroupSection key={item.id} group={item} activePath={activePath} />
              ) : (
                <NavItemRow key={item.id} item={item} activePath={activePath} />
              ),
            )}
          </ModuleSection>
        );
      })}
    </div>
  );
}
