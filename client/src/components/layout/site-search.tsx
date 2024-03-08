/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
import { useUserPermissions } from "@/context/user-permissions";
import { useUserFavorites } from "@/hooks/useQueries";
import { upperFirst } from "@/lib/utils";
import { RouteObjectWithPermission, routes } from "@/routing/AppRoutes";
import { useHeaderStore } from "@/stores/HeaderStore";
import { type UserFavorite } from "@/types/accounts";
import { faCircleExclamation } from "@fortawesome/pro-duotone-svg-icons";
import { faCommand } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import React from "react";
import { useNavigate } from "react-router-dom";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "../ui/command";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";

const prepareRouteGroups = (routeList: typeof routes) => {
  return routeList.reduce(
    (acc, route) => {
      if (!acc[route.group]) {
        acc[route.group] = [];
      }
      acc[route.group].push(route);

      return acc;
    },
    {} as Record<string, typeof routes>,
  );
};

export function SiteSearchInput() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            aria-label="Open site search"
            aria-expanded={useHeaderStore.get("searchDialogOpen")}
            onClick={() => useHeaderStore.set("searchDialogOpen", true)}
            className="group hidden h-8 w-[250px] items-center justify-between rounded-md border border-muted-foreground/20 bg-secondary px-3 py-2 text-sm hover:border-muted-foreground/80 hover:bg-accent lg:flex"
          >
            <div className="flex items-center">
              <MagnifyingGlassIcon className="mr-2 size-5 text-muted-foreground group-hover:text-foreground" />
              <span className="text-muted-foreground">Search...</span>
            </div>
            <kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-x-1 rounded border border-border bg-background px-1.5 font-mono text-[10px] font-medium text-foreground opacity-100">
              <FontAwesomeIcon icon={faCommand} className="mb-0.5" />
              <span className="text-xs">K</span>
            </kbd>
          </button>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Site Search</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function SiteSearch() {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const [open, setOpen] = useHeaderStore.use("searchDialogOpen");
  const [searchText, setSearchText] = React.useState<string>("");
  const { data: userFavorites } = useUserFavorites();

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const favoritePaths = new Set(
    (userFavorites as unknown as UserFavorite[])?.map(
      (favorite) => favorite.page,
    ),
  );

  const filteredRoutes = routes.filter((route) => {
    if (route.excludeFromMenu) return false;
    if (route.path.endsWith("/:id")) return false;
    if (route.permission && !userHasPermission(route.permission)) return false;
    return isAuthenticated;
  });

  // Partitioning routes into favorite and other routes
  const favoriteRoutes = filteredRoutes.filter((route) =>
    favoritePaths.has(route.path),
  );
  const otherRoutes = filteredRoutes.filter(
    (route) => !favoritePaths.has(route.path),
  );

  const groupedRoutes = prepareRouteGroups(otherRoutes); // Prepare groups only for non-favorite routes

  // Prepare favorite commands for rendering
  const favoriteCommands = favoriteRoutes.filter((route) =>
    route.title.toLowerCase().includes(searchText.toLowerCase()),
  );

  const filteredGroups = Object.entries(groupedRoutes).reduce(
    (acc, [group, groupRoutes]) => {
      const filtered = groupRoutes.filter((route) =>
        route.title.toLowerCase().includes(searchText.toLowerCase()),
      );

      if (filtered.length) {
        acc[group] = filtered;
      }

      return acc;
    },
    {} as Record<string, RouteObjectWithPermission[]>,
  );

  const handleDialogOpenChange = (isOpen: boolean) => {
    setOpen(isOpen);
    if (!isOpen) {
      setSearchText("");
    }
  };

  return (
    <CommandDialog open={open} onOpenChange={handleDialogOpenChange}>
      <CommandInput
        placeholder="Search..."
        value={searchText}
        onValueChange={setSearchText}
      />
      <CommandList>
        {/* Render favorite commands first */}
        {favoriteCommands.length > 0 && (
          <React.Fragment key="favorites">
            <CommandGroup heading="Favorites">
              {favoriteCommands.map((cmd) => (
                <CommandItem
                  key={cmd.path + "-favorite-item"}
                  onSelect={() => {
                    navigate(cmd.path);
                    setOpen(false);
                  }}
                  className="text-mono hover:cursor-pointer"
                >
                  {cmd.title}
                </CommandItem>
              ))}
            </CommandGroup>
            <CommandSeparator key="favorites-separator" />
          </React.Fragment>
        )}
        {/* Render other groups */}
        {Object.keys(filteredGroups).length === 0 &&
          favoriteCommands.length === 0 && (
            <CommandEmpty key="empty">
              <FontAwesomeIcon
                icon={faCircleExclamation}
                className="mx-auto size-6 text-accent-foreground"
              />
              <p className="mt-4 font-semibold text-accent-foreground">
                No results found
              </p>
              <p className="mt-2 text-muted-foreground">
                No pages found for this search term. Please try again.
              </p>
            </CommandEmpty>
          )}
        {Object.entries(filteredGroups).map(([group, groupCommands]) => (
          <React.Fragment key={group}>
            <CommandGroup heading={upperFirst(group)}>
              {groupCommands.map((cmd) => (
                <CommandItem
                  key={cmd.path + "-group-item"}
                  onSelect={() => {
                    navigate(cmd.path);
                    setOpen(false);
                  }}
                  className="text-mono hover:cursor-pointer"
                >
                  {cmd.title}
                </CommandItem>
              ))}
            </CommandGroup>
            <CommandSeparator key={group + "-separator"} />
          </React.Fragment>
        ))}
      </CommandList>
      <div className="sticky flex justify-center space-x-1 border-t bg-background py-2">
        <span className="text-xs">&#8593;</span>
        <span className="text-xs">&#8595;</span>
        <p className="pr-2 text-xs">to navigate</p>
        <span className="text-xs">&#x23CE;</span>
        <p className="pr-2 text-xs">to select</p>
        <span className="text-xs">esc</span>
        <p className="text-xs">to close</p>
      </div>
    </CommandDialog>
  );
}
