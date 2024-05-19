import { useUserPermissions } from "@/context/user-permissions";
import { useUserFavorites } from "@/hooks/useQueries";
import { upperFirst } from "@/lib/utils";
import { RouteObjectWithPermission, routes } from "@/routing/AppRoutes";
import { useHeaderStore } from "@/stores/HeaderStore";
import { type UserFavorite } from "@/types/accounts";
import { faPage, faStar } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  LaptopIcon,
  MagnifyingGlassIcon,
  MoonIcon,
  SunIcon,
} from "@radix-ui/react-icons";
import { VariantProps } from "class-variance-authority";
import { AlertCircleIcon } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router-dom";
import { Badge, badgeVariants } from "../ui/badge";
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
import { KeyCombo, Keys, ShortcutsProvider } from "../ui/keyboard";
import { useTheme } from "../ui/theme-provider";
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
          <span
            aria-label="Open site search"
            aria-expanded={useHeaderStore.get("searchDialogOpen")}
            onClick={() => useHeaderStore.set("searchDialogOpen", true)}
            className="border-muted-foreground/20 hover:border-muted-foreground/80 hover:bg-accent group hidden h-8 w-[250px] items-center justify-between rounded-md border px-3 py-2 text-sm hover:cursor-pointer xl:flex"
          >
            <div className="flex items-center">
              <MagnifyingGlassIcon className="text-muted-foreground group-hover:text-foreground mr-2 size-5" />
              <span className="text-muted-foreground">Search...</span>
            </div>
            <div className="pointer-events-none inline-flex select-none">
              <ShortcutsProvider os="mac">
                <KeyCombo keyNames={[Keys.Command, "K"]} />
              </ShortcutsProvider>
            </div>
          </span>
        </TooltipTrigger>
        <TooltipContent side="left" sideOffset={5}>
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
  const { setTheme } = useTheme();

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || e.key === "/") {
        if (
          (e.target instanceof HTMLElement && e.target.isContentEditable) ||
          e.target instanceof HTMLInputElement ||
          e.target instanceof HTMLTextAreaElement ||
          e.target instanceof HTMLSelectElement
        ) {
          return;
        }

        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [setOpen]);

  const runCommand = React.useCallback(
    (command: () => unknown) => {
      setOpen(false);
      command();
    },
    [setOpen],
  );

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

  type BadgeVariant = VariantProps<typeof badgeVariants>["variant"];

  const groupVariants: Record<string, BadgeVariant>[] = [
    { administration: "inactive" },
    { main: "active" },
    { billing: "purple" },
    { accounting: "warning" },
    { dispatch: "info" },
    { equipment: "purple" },
  ];

  const mapGroupVariant = (group: string) => {
    const variant = groupVariants.find((v) => v[group]);
    return variant ? variant[group] : "default";
  };

  // Upper case the first letter of the group name
  const upperFirstGroup = (group: string) => upperFirst(group);

  return (
    <CommandDialog open={open} onOpenChange={handleDialogOpenChange}>
      <CommandInput
        placeholder="What do you need?"
        value={searchText}
        onValueChange={setSearchText}
      />
      <CommandList>
        {favoriteCommands.length > 0 && (
          <CommandGroup heading="Favorites" key="favorites">
            {favoriteCommands.map((cmd) => (
              <CommandItem
                key={cmd.path + "-favorite-item"}
                onSelect={() => {
                  runCommand(() => navigate(cmd.path as string));
                }}
              >
                <FontAwesomeIcon icon={faStar} className="mr-2 size-4" />
                {cmd.title}
                <Badge variant={mapGroupVariant(cmd.group)} className="ml-auto">
                  {upperFirstGroup(cmd.group)}
                </Badge>
              </CommandItem>
            ))}
          </CommandGroup>
        )}
        <CommandSeparator key="favorites-separator" />
        {Object.keys(filteredGroups).length === 0 &&
          favoriteCommands.length === 0 && (
            <CommandEmpty key="empty">
              <AlertCircleIcon className="text-accent-foreground mx-auto size-6" />
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
                    runCommand(() => navigate(cmd.path as string));
                  }}
                >
                  <FontAwesomeIcon icon={faPage} className="mr-2 size-4" />
                  {cmd.title}
                </CommandItem>
              ))}
            </CommandGroup>
            <CommandSeparator key={group + "-separator"} />
          </React.Fragment>
        ))}
        <CommandSeparator />
        <CommandGroup heading="Theme">
          <CommandItem onSelect={() => runCommand(() => setTheme("light"))}>
            <SunIcon className="mr-2 size-4" />
            Light
          </CommandItem>
          <CommandItem onSelect={() => runCommand(() => setTheme("dark"))}>
            <MoonIcon className="mr-2 size-4" />
            Dark
          </CommandItem>
          <CommandItem onSelect={() => runCommand(() => setTheme("system"))}>
            <LaptopIcon className="mr-2 size-4" />
            System
          </CommandItem>
        </CommandGroup>
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
