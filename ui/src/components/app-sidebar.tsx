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
import { usePermissions } from "@/hooks/use-permission";
import { routes } from "@/lib/nav-links";
import { cn } from "@/lib/utils";
import { useIsAuthenticated, useUser } from "@/stores/user-store";
import { Resource } from "@/types/audit-entry";
import { RouteInfo } from "@/types/nav-links";
import { Action } from "@/types/roles-permissions";
import { faSearch } from "@fortawesome/pro-regular-svg-icons";
import { ChevronRightIcon } from "@radix-ui/react-icons";
import React, { memo, useEffect, useMemo, useRef, useState } from "react";
import { isMacOs } from "react-device-detect";
import { Link, useLocation } from "react-router";
import { FavoritesSidebar } from "./favorites-sidebar";
import { NavUser } from "./nav-user";
import { OrganizationSwitcher } from "./organization-switcher";
import { SearchDialog } from "./site-search/site-search-dialog";
import Highlight from "./ui/highlight";
import { Icon } from "./ui/icons";
import { Kbd } from "./ui/kbd";
import { Skeleton } from "./ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";

const isRouteActive = (currentPath: string, itemPath?: string): boolean => {
  if (!itemPath) return false;

  if (itemPath === "/") {
    return currentPath === "/";
  }

  const normalizedCurrentPath = currentPath.endsWith("/")
    ? currentPath.slice(0, -1)
    : currentPath;
  const normalizedItemPath = itemPath.endsWith("/")
    ? itemPath.slice(0, -1)
    : itemPath;

  return normalizedCurrentPath.startsWith(normalizedItemPath);
};

const hasActiveChild = (currentPath: string, item: RouteInfo): boolean => {
  if (isRouteActive(currentPath, item.link)) return true;
  if (!item.tree) return false;

  return item.tree.some(
    (subItem) =>
      isRouteActive(currentPath, subItem.link) ||
      hasActiveChild(currentPath, subItem),
  );
};

const filterRoutesByPermission = (
  routes: RouteInfo[],
  can: (resource: Resource, action: Action) => boolean,
): RouteInfo[] => {
  return routes
    .map((route) => {
      if (route.tree) {
        const filteredChildren = filterRoutesByPermission(route.tree, can);

        if (filteredChildren.length > 0) {
          return {
            ...route,
            tree: filteredChildren,
          };
        }
        return null;
      }

      if (can(route.key as Resource, "read" as Action)) {
        return route;
      }

      return null;
    })
    .filter((route): route is RouteInfo => route !== null);
};

const filterRoutesBySearch = (
  routes: RouteInfo[],
  searchQuery: string,
): RouteInfo[] => {
  if (!searchQuery.trim()) return routes;

  const query = searchQuery.toLowerCase().trim();
  const queryWords = query.split(/\s+/);

  return routes
    .map((route) => {
      const routeLabel = route.label.toLowerCase();
      const routeMatches = queryWords.every((word) =>
        routeLabel.includes(word),
      );

      if (route.tree) {
        const filteredChildren = filterRoutesBySearch(route.tree, searchQuery);

        if (routeMatches || filteredChildren.length > 0) {
          return {
            ...route,
            tree: filteredChildren,
          };
        }
        return null;
      }

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
  isNested?: boolean;
}) {
  const isActive = isRouteActive(currentPath, item.link);
  const hasActive = hasActiveChild(currentPath, item);

  const hasChildrenDuringSearch = !!(
    searchQuery &&
    item.tree &&
    item.tree.length > 0
  );

  const [isOpen, setIsOpen] = React.useState(
    hasActive || hasChildrenDuringSearch,
  );

  const handleNavigation = React.useCallback(
    (
      e: React.MouseEvent<HTMLAnchorElement>,
      targetPath: string | undefined,
    ) => {
      if (!targetPath || targetPath === "#") return;

      if (isRouteActive(currentPath, targetPath)) {
        e.preventDefault();
        return;
      }
    },
    [currentPath],
  );

  React.useEffect(() => {
    if (hasActive) {
      setIsOpen(true);
    }
  }, [hasActive, currentPath]);

  React.useEffect(() => {
    if (searchQuery && item.tree && item.tree.length > 0) {
      setIsOpen(true);
    } else if (!searchQuery && !hasActive) {
      setIsOpen(false);
    }
  }, [searchQuery, item.tree, hasActive]);

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
        <Collapsible
          open={isOpen}
          onOpenChange={(open) => setIsOpen(open)}
          className="min-w-full"
        >
          <SidebarMenuButton className="w-full cursor-default" asChild>
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
                "data-[state=open]:rotate-90 mt-0.5 cursor-pointer",
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

  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  const filteredRoutes = useMemo(() => {
    if (!user) return [];
    const permissionFilteredRoutes = filterRoutesByPermission(routes, can);
    return filterRoutesBySearch(permissionFilteredRoutes, debouncedSearchQuery);
  }, [can, user, debouncedSearchQuery]);

  const isLoading = isAuthenticated && !user;

  return (
    <Sidebar variant="floating" {...props}>
      <SidebarHeader className="gap-4">
        <OrganizationSwitcher />
        <SearchDialog />
        <SiteInputWrapper
          searchQuery={searchQuery}
          setSearchQuery={setSearchQuery}
        />
      </SidebarHeader>
      <SidebarContent>
        <FavoritesSidebar />
        <SidebarGroup>
          <SidebarGroupLabel className="font-semibold uppercase select-none">
            Navigation
          </SidebarGroupLabel>
          <SidebarGroupContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(10)].map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : filteredRoutes.length === 0 && debouncedSearchQuery ? (
              <div className="px-2 py-8 text-center text-sm text-muted-foreground">
                No navigation items found for &ldquo;{debouncedSearchQuery}
                &rdquo;
              </div>
            ) : (
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

function SiteInputWrapper({
  searchQuery,
  setSearchQuery,
}: {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
}) {
  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
      if (e.key === "Escape" && searchQuery) {
        setSearchQuery("");
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [searchQuery, setSearchQuery]);

  return (
    <div className="relative">
      <Icon
        icon={faSearch}
        className="absolute top-1/2 left-2 size-4 -translate-y-1/2 text-muted-foreground"
      />
      <SidebarInput
        ref={searchInputRef}
        placeholder="Search navigation..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        className="w-full"
        icon={<Icon icon={faSearch} className="size-3 text-muted-foreground" />}
        rightElement={
          <Tooltip>
            <TooltipTrigger asChild>
              <Kbd aria-label="Meta">{isMacOs ? "⌘" : "Ctrl"} + K</Kbd>
            </TooltipTrigger>
            <TooltipContent className="flex items-center gap-2 text-xs">
              <Kbd>{isMacOs ? "⌘" : "Ctrl"} + K</Kbd>
              <p>to trigger advanced search</p>
            </TooltipContent>
          </Tooltip>
        }
      />
    </div>
  );
}
