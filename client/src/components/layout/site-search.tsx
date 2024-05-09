import { useUserPermissions } from "@/context/user-permissions";
import { useUserFavorites } from "@/hooks/useQueries";
import { upperFirst } from "@/lib/utils";
import { RouteObjectWithPermission, routes } from "@/routing/AppRoutes";
import { useHeaderStore } from "@/stores/HeaderStore";
import { type UserFavorite } from "@/types/accounts";
import { faCommand } from "@fortawesome/pro-regular-svg-icons";
import { faCircleExclamation } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import React from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../ui/button";
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

export function SearchButton() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            variant="outline"
            aria-label="Open site search"
            aria-expanded={useHeaderStore.get("searchDialogOpen")}
            onClick={() => useHeaderStore.set("searchDialogOpen", true)}
            className="border-muted-foreground/40 hover:border-muted-foreground/80 group relative flex size-8 xl:hidden"
          >
            <MagnifyingGlassIcon className="text-muted-foreground group-hover:text-foreground size-5" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Site Search</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function SiteSearchInput() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            aria-label="Open site search"
            aria-expanded={useHeaderStore.get("searchDialogOpen")}
            onClick={() => useHeaderStore.set("searchDialogOpen", true)}
            className="border-muted-foreground/20 hover:border-muted-foreground/80 hover:bg-accent group hidden h-8 w-[250px] items-center justify-between rounded-md border px-3 py-2 text-sm xl:flex" // Adjusted for responsiveness
          >
            <div className="flex items-center">
              <MagnifyingGlassIcon className="text-muted-foreground group-hover:text-foreground mr-2 size-5" />
              <span className="text-muted-foreground">Search...</span>
            </div>
            <kbd className="border-border bg-background text-foreground pointer-events-none inline-flex h-5 select-none items-center gap-x-1 rounded border px-1.5 font-mono text-[10px] font-medium opacity-100">
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
      (favorite) => favorite.pageLink,
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
                className="text-accent-foreground mx-auto size-6"
              />
              <p className="text-accent-foreground mt-4 font-semibold">
                No results found
              </p>
              <p className="text-muted-foreground mt-2">
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
      <div className="bg-background sticky flex justify-center space-x-1 border-t py-2">
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
