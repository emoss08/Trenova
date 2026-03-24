import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import { isNavGroup, type NavGroup, type NavItem, type NavModule } from "@/config/navigation.types";
import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { BetaTag } from "../beta-tag";

function isRouteActive(currentPath: string, itemPath?: string): boolean {
  if (!itemPath) return false;
  if (itemPath === "/") return currentPath === "/";

  const normalizedCurrentPath = currentPath.endsWith("/") ? currentPath.slice(0, -1) : currentPath;
  const normalizedItemPath = itemPath.endsWith("/") ? itemPath.slice(0, -1) : itemPath;

  return normalizedCurrentPath.startsWith(normalizedItemPath);
}

function hasActiveChild(currentPath: string, items: (NavItem | NavGroup)[]): boolean {
  return items.some((item) => {
    if (isNavGroup(item)) {
      return hasActiveChild(currentPath, item.items);
    }
    return isRouteActive(currentPath, item.path);
  });
}

function hasActiveInModule(currentPath: string, module: NavModule): boolean {
  if (isRouteActive(currentPath, module.basePath)) return true;
  return hasActiveChild(currentPath, module.navigation);
}

interface SubItemProps {
  item: NavItem | NavGroup;
  currentPath: string;
}

function SubItem({ item, currentPath }: SubItemProps) {
  if (isNavGroup(item)) {
    return <SubGroup group={item} currentPath={currentPath} />;
  }

  const isActive = isRouteActive(currentPath, item.path);

  return (
    <SidebarMenuSubItem>
      <SidebarMenuSubButton render={<Link to={item.path} />} isActive={isActive}>
        <span className="truncate">{item.label}</span> {item.includeBetaTag && <BetaTag />}
      </SidebarMenuSubButton>
    </SidebarMenuSubItem>
  );
}

interface SubGroupProps {
  group: NavGroup;
  currentPath: string;
}

function SubGroup({ group, currentPath }: SubGroupProps) {
  const hasActive = hasActiveChild(currentPath, group.items);
  const [isOpen, setIsOpen] = useState(group.defaultOpen ?? hasActive);
  const isExpanded = hasActive || isOpen;

  return (
    <SidebarMenuSubItem>
      <Collapsible open={isExpanded} onOpenChange={setIsOpen} className="w-full">
        <CollapsibleTrigger
          render={
            <button
              className={cn(
                "group/sub-item flex h-8 w-full items-center justify-between rounded-md px-2 text-sm",
                "cursor-pointer text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                "transition-colors",
                hasActive && "bg-sidebar-accent font-medium text-sidebar-accent-foreground",
              )}
            />
          }
        >
          <span>{group.label}</span>
          <span className="grid size-5 flex-none place-content-center rounded-sm group-hover/sub-item:bg-primary/10">
            <ChevronRight
              className={cn("size-4 transition-transform duration-200", isExpanded && "rotate-90")}
            />
          </span>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <SidebarMenuSub>
            {group.items.map((subItem) => (
              <SubItem key={subItem.id} item={subItem} currentPath={currentPath} />
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    </SidebarMenuSubItem>
  );
}

interface TreeProps {
  module: NavModule;
  currentPath: string;
}

function Tree({ module, currentPath }: TreeProps) {
  const isActive = isRouteActive(currentPath, module.basePath);
  const hasActive = hasActiveInModule(currentPath, module);

  const hasChildren = !module.hideSecondarySidebar && module.navigation.length > 0;

  const [isOpen, setIsOpen] = useState(hasActive);
  const isExpanded = hasActive || isOpen;

  if (!hasChildren) {
    return (
      <SidebarMenuItem className="cursor-pointer">
        <SidebarMenuButton render={<Link to={module.basePath} />} isActive={isActive}>
          <module.icon
            className={cn(
              "size-4 text-sidebar-primary/40 group-hover/menu-button:text-sidebar-accent-foreground",
              isActive && "text-sidebar-accent-foreground",
            )}
          />
          <span>{module.label}</span>
        </SidebarMenuButton>
      </SidebarMenuItem>
    );
  }

  return (
    <SidebarMenuItem className="cursor-pointer">
      <Collapsible open={isExpanded} onOpenChange={setIsOpen} className="w-full">
        <CollapsibleTrigger
          render={<SidebarMenuButton isActive={hasActive} className="cursor-pointer" />}
        >
          <module.icon
            className={cn(
              "size-4 text-sidebar-primary/40 group-hover/menu-button:text-sidebar-accent-foreground",
              hasActive && "text-sidebar-accent-foreground",
            )}
          />
          <span className="flex-1 truncate">{module.label}</span>
          <span className="grid size-5 flex-none place-content-center rounded-sm group-hover/menu-button:bg-primary/10">
            <ChevronRight
              className={cn("size-4 transition-transform duration-200", isExpanded && "rotate-90")}
            />
          </span>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <SidebarMenuSub>
            {module.navigation.map((item) => (
              <SubItem key={item.id} item={item} currentPath={currentPath} />
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    </SidebarMenuItem>
  );
}

interface SidebarNavProps {
  modules: NavModule[];
}

export function SidebarNav({ modules }: SidebarNavProps) {
  const location = useLocation();
  const currentPath = location.pathname;

  return (
    <SidebarMenu className="gap-1">
      {modules.map((module) => (
        <Tree key={module.id} module={module} currentPath={currentPath} />
      ))}
    </SidebarMenu>
  );
}
