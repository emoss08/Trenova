/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import React, { memo, useEffect, useMemo, useRef, useState } from "react";

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
  SidebarInput,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import { useDebounce } from "@/hooks/use-debounce";
import { usePermissions } from "@/hooks/use-permissions";
import { routes } from "@/lib/nav-links";
import { cn } from "@/lib/utils";
import { useIsAuthenticated, useUser } from "@/stores/user-store";
import { Resource } from "@/types/audit-entry";
import { RouteInfo } from "@/types/nav-links";
import { Action } from "@/types/roles-permissions";
import { faSearch, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { ChevronRightIcon } from "@radix-ui/react-icons";
import { isMacOs } from "react-device-detect";
import { Link, useLocation } from "react-router";
import { FavoritesSidebar } from "./favorites-sidebar";
import { NavUser } from "./nav-user";
import { OrganizationSwitcher } from "./organization-switcher";
import Highlight from "./ui/highlight";
import { Icon } from "./ui/icons";
import { Kbd, KbdKey } from "./ui/kibo-ui/kbd";
import { Skeleton } from "./ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";
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
const hasActiveChild = (currentPath: string, item: RouteInfo): boolean => {
  if (isRouteActive(currentPath, item.link)) return true;
  if (!item.tree) return false;

  return item.tree.some(
    (subItem) =>
      isRouteActive(currentPath, subItem.link) ||
      hasActiveChild(currentPath, subItem),
  );
};

// Filter routes based on permissions
const filterRoutesByPermission = (
  routes: RouteInfo[],
  can: (resource: Resource, action: Action) => boolean,
): RouteInfo[] => {
  return routes
    .map((route) => {
      // If route has children, filter them first
      if (route.tree) {
        const filteredChildren = filterRoutesByPermission(route.tree, can);

        // If this is a grouping node (like ConfigurationFiles) and has accessible children, include it
        if (filteredChildren.length > 0) {
          return {
            ...route,
            tree: filteredChildren,
          };
        }
        // If no children are accessible, exclude this route
        return null;
      }

      // For leaf nodes, check if user has permission
      if (can(route.key as Resource, "read" as Action)) {
        return route;
      }

      return null;
    })
    .filter((route): route is RouteInfo => route !== null);
};

// Filter routes based on search query
const filterRoutesBySearch = (
  routes: RouteInfo[],
  searchQuery: string,
): RouteInfo[] => {
  if (!searchQuery.trim()) return routes;

  const query = searchQuery.toLowerCase().trim();
  const queryWords = query.split(/\s+/);

  return routes
    .map((route) => {
      // Check if current route matches all query words
      const routeLabel = route.label.toLowerCase();
      const routeMatches = queryWords.every((word) =>
        routeLabel.includes(word),
      );

      // If route has children, filter them
      if (route.tree) {
        const filteredChildren = filterRoutesBySearch(route.tree, searchQuery);

        // Include parent if it matches or has matching children
        if (routeMatches || filteredChildren.length > 0) {
          return {
            ...route,
            tree: filteredChildren,
          };
        }
        return null;
      }

      // For leaf nodes, include if matches
      return routeMatches ? route : null;
    })
    .filter((route): route is RouteInfo => route !== null);
};

function Tree({
  item,
  currentPath,
  searchQuery,
}: {
  item: RouteInfo;
  currentPath: string;
  searchQuery?: string;
}) {
  const isActive = isRouteActive(currentPath, item.link);
  const hasActive = hasActiveChild(currentPath, item);
  const [isOpen, setIsOpen] = React.useState(hasActive);

  // Helper to handle navigation - prevent navigation if already on the same route
  const handleNavigation = React.useCallback(
    (
      e: React.MouseEvent<HTMLAnchorElement>,
      targetPath: string | undefined,
    ) => {
      if (!targetPath || targetPath === "#") return;

      // If we're already on this route, prevent navigation to preserve query params
      if (isRouteActive(currentPath, targetPath)) {
        e.preventDefault();
        return;
      }

      // Otherwise, allow normal navigation (clears query params for different tables)
    },
    [currentPath],
  );

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
              onClick={(e) => handleNavigation(e, item.link)}
            >
              {item.icon && (
                <Icon
                  icon={item.icon}
                  className={isActive ? "text-sidebar-accent-foreground" : ""}
                />
              )}
              <span className="flex-1">
                {searchQuery ? (
                  <Highlight text={item.label} highlight={searchQuery} />
                ) : (
                  item.label
                )}
              </span>
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
            <Link
              to={item.link || "#"}
              onClick={(e) => handleNavigation(e, item.link)}
            >
              {item.icon && <Icon icon={item.icon} />}
              <span>
                {searchQuery ? (
                  <Highlight text={item.label} highlight={searchQuery} />
                ) : (
                  item.label
                )}
              </span>
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
                    <Tree
                      item={subItem}
                      currentPath={currentPath}
                      searchQuery={searchQuery}
                    />
                  ) : (
                    <SidebarMenuSubButton
                      asChild
                      isActive={isRouteActive(currentPath, subItem.link)}
                    >
                      <Link
                        to={subItem.link || "#"}
                        onClick={(e) => handleNavigation(e, subItem.link)}
                        className="flex items-center gap-2"
                      >
                        {subItem.icon && <Icon icon={subItem.icon} />}
                        <span>
                          {searchQuery ? (
                            <Highlight
                              text={subItem.label}
                              highlight={searchQuery}
                            />
                          ) : (
                            subItem.label
                          )}
                        </span>
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

export const AppSidebar = memo(function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar>) {
  const location = useLocation();
  const currentPath = location.pathname;
  const { can } = usePermissions();
  const isAuthenticated = useIsAuthenticated();
  const user = useUser();
  const [searchQuery, setSearchQuery] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);
  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  const filteredRoutes = useMemo(() => {
    if (!user) return [];
    const permissionFilteredRoutes = filterRoutesByPermission(routes, can);
    return filterRoutesBySearch(permissionFilteredRoutes, debouncedSearchQuery);
  }, [can, user, debouncedSearchQuery]);

  // Show loading state while user data is being fetched
  const isLoading = isAuthenticated && !user;

  // Add keyboard shortcut for search
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + K to focus search
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
      // Escape to clear search
      if (e.key === "Escape" && searchQuery) {
        setSearchQuery("");
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [searchQuery]);

  // TODO(wolfred): Allow for the sidebar to be configurable by the user.
  return (
    <Sidebar variant="floating" {...props}>
      <SidebarHeader className="gap-4">
        <OrganizationSwitcher />
        <WorkflowPlaceholder />
        <div className="relative">
          <Icon
            icon={faSearch}
            className="absolute left-2 top-1/2 -translate-y-1/2 size-4 text-muted-foreground"
          />
          <SidebarInput
            ref={searchInputRef}
            placeholder="Search navigation..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full"
            icon={
              <Icon icon={faSearch} className="size-3 text-muted-foreground" />
            }
            rightElement={
              <Tooltip>
                <TooltipTrigger>
                  <Kbd>
                    <KbdKey aria-label="Meta">{isMacOs ? "⌘" : "Ctrl"}</KbdKey>
                    <KbdKey>K</KbdKey>
                  </Kbd>
                </TooltipTrigger>
                <TooltipContent>
                  <p>{isMacOs ? "⌘" : "Ctrl"} + K to trigger advanced search</p>
                </TooltipContent>
              </Tooltip>
            }
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              aria-label="Clear search"
            >
              <Icon icon={faXmark} className="size-3" />
            </button>
          )}
        </div>
      </SidebarHeader>
      <SidebarContent>
        <FavoritesSidebar />
        <SidebarGroup>
          <SidebarGroupLabel className="select-none font-semibold uppercase">
            Navigation
          </SidebarGroupLabel>
          <SidebarGroupContent>
            {isLoading ? (
              // Show loading skeletons while permissions are loading
              <div className="space-y-2">
                {[...Array(5)].map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : filteredRoutes.length === 0 && debouncedSearchQuery ? (
              // Show no results message
              <div className="px-2 py-8 text-center text-sm text-muted-foreground">
                No navigation items found for &ldquo;{debouncedSearchQuery}
                &rdquo;
              </div>
            ) : (
              // Show navigation items once loaded
              filteredRoutes.map((item) => (
                <Tree
                  key={item.key}
                  item={item}
                  currentPath={currentPath}
                  searchQuery={debouncedSearchQuery}
                />
              ))
            )}
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  );
});
