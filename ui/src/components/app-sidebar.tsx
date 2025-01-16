import * as React from "react";

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import { routes } from "@/lib/nav-links";
import { cn } from "@/lib/utils";
import { routeInfo } from "@/types/nav-links";
import { ChevronRightIcon } from "@radix-ui/react-icons";
import { Link, useLocation } from "react-router";
import { NavUser } from "./nav-user";
import { OrganizationSwitcher } from "./organization-switcher";
import { SearchForm } from "./site-search";
import { Icon } from "./ui/icons";
import { WorkflowPlaceholder } from "./workflow";

// Helper function to check if a route is active
const isRouteActive = (currentPath: string, itemPath?: string): boolean => {
  if (!itemPath) return false;

  // Special handling for root path
  if (itemPath === "/") {
    return currentPath === "/";
  }

  // For other paths, ensure we match complete segments
  // This prevents "/teams" from matching "/team"
  const normalizedCurrentPath = currentPath.endsWith("/")
    ? currentPath.slice(0, -1)
    : currentPath;
  const normalizedItemPath = itemPath.endsWith("/")
    ? itemPath.slice(0, -1)
    : itemPath;

  return normalizedCurrentPath.startsWith(normalizedItemPath);
};

// Helper function to check if any child route is active
const hasActiveChild = (currentPath: string, item: routeInfo): boolean => {
  if (isRouteActive(currentPath, item.link)) return true;
  if (!item.tree) return false;

  return item.tree.some(
    (subItem) =>
      isRouteActive(currentPath, subItem.link) ||
      hasActiveChild(currentPath, subItem),
  );
};

function Tree({ item, currentPath }: { item: routeInfo; currentPath: string }) {
  const isActive = isRouteActive(currentPath, item.link);
  const hasActive = hasActiveChild(currentPath, item);
  const [isOpen, setIsOpen] = React.useState(hasActive);

  // Update open state when active state changes
  React.useEffect(() => {
    if (hasActive) {
      setIsOpen(true);
    }
  }, [hasActive, currentPath]);

  if (!item.tree) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton asChild data-active={isActive} isActive={isActive}>
            <Link
              to={item.link || "#"}
              className="data-[active=true]:border data-[active=true]:border-input"
            >
              {item.icon && (
                <Icon
                  icon={item.icon}
                  className={isActive ? "text-sidebar-accent-foreground" : ""}
                />
              )}
              <span className="flex-1">{item.label}</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    );
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem className="items-center">
        <Collapsible open={isOpen} onOpenChange={(open) => setIsOpen(open)}>
          <SidebarMenuButton className="cursor-default" asChild>
            <Link to={item.link || "#"}>
              {item.icon && <Icon icon={item.icon} />}
              <span>{item.label}</span>
            </Link>
          </SidebarMenuButton>
          <CollapsibleTrigger asChild>
            <SidebarMenuAction
              className={cn(
                "data-[state=open]:rotate-90 mt-0.5",
                hasActive ? "bg-sidebar-accent" : "",
              )}
            >
              <ChevronRightIcon className="size-4" />
              <span className="sr-only">Toggle</span>
            </SidebarMenuAction>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <SidebarMenuSub>
              {item.tree.map((subItem) => (
                <SidebarMenuSubItem key={subItem.key}>
                  {subItem.tree ? (
                    <Tree item={subItem} currentPath={currentPath} />
                  ) : (
                    <SidebarMenuSubButton
                      asChild
                      isActive={isRouteActive(currentPath, subItem.link)}
                    >
                      <Link
                        to={subItem.link || "#"}
                        className="flex items-center gap-2"
                      >
                        {subItem.icon && <Icon icon={subItem.icon} />}
                        <span>{subItem.label}</span>
                      </Link>
                    </SidebarMenuSubButton>
                  )}
                </SidebarMenuSubItem>
              ))}
            </SidebarMenuSub>
          </CollapsibleContent>
        </Collapsible>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const location = useLocation();
  const currentPath = location.pathname;

  // TODO(wolfred): Allow for the sidebar to be configurable by the user.
  return (
    <Sidebar variant="floating" {...props}>
      <SidebarHeader className="gap-4">
        <OrganizationSwitcher />
        <SearchForm />
        <WorkflowPlaceholder />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel className="select-none font-semibold uppercase">
            Navigation
          </SidebarGroupLabel>
          <SidebarGroupContent>
            {routes.map((item) => (
              <Tree key={item.key} item={item} currentPath={currentPath} />
            ))}
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  );
}
