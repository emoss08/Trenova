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
import { upperFirst } from "@/lib/utils";
import { routes } from "@/routing/AppRoutes";
import { useHeaderStore } from "@/stores/HeaderStore";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { AlertCircle } from "lucide-react";
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
            className="group flex h-8 w-[250px] items-center justify-between rounded-md border border-foreground/20 bg-secondary px-3 py-2 text-sm hover:border-muted-foreground/80 hover:bg-accent md:flex"
          >
            <div className="flex items-center">
              <MagnifyingGlassIcon className="mr-2 size-5 text-muted-foreground group-hover:text-foreground" />
              <span className="text-muted-foreground">Search...</span>
            </div>
            <kbd className="pointer-events-none inline-flex h-5 select-none items-center rounded border border-border bg-background px-1.5 font-mono text-[10px] font-medium text-foreground opacity-100">
              <span className="text-xs">⌘ K</span>
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

  // Filtering and preparing route groups
  const filteredRoutes = routes.filter((route) => {
    if (route.excludeFromMenu) return false;
    if (route.path.endsWith("/:id")) return false;
    if (route.permission && !userHasPermission(route.permission)) return false;
    return isAuthenticated;
  });

  const groupedRoutes = prepareRouteGroups(filteredRoutes);

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
    {} as Record<string, typeof routes>,
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
        {Object.entries(filteredGroups).length === 0 && (
          <CommandEmpty>
            <AlertCircle className="mx-auto size-6 text-accent-foreground" />
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
                  key={cmd.path + "-item"}
                  onSelect={() => {
                    navigate(cmd.path);
                    setOpen(false);
                  }}
                  className="hover:cursor-pointer"
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
