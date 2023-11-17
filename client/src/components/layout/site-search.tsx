/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
} from "../ui/command";

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

export function SiteSearch() {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const [open, setOpen] = React.useState(false);
  const [searchText, setSearchText] = React.useState("");

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

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput
        placeholder="Search..."
        value={searchText}
        onValueChange={setSearchText}
      />
      <CommandList>
        {Object.entries(filteredGroups).length === 0 && (
          <CommandEmpty>
            <AlertCircle className="mx-auto h-6 w-6 text-accent-foreground" />
            <p className="mt-4 font-semibold text-accent-foreground">
              No results found
            </p>
            <p className="mt-2 text-muted-foreground">
              No pages found for this search term. Please try again.
            </p>
          </CommandEmpty>
        )}
        {Object.entries(filteredGroups).map(([group, groupCommands]) => (
          <CommandGroup key={group} heading={upperFirst(group)}>
            {groupCommands.map((cmd) => (
              <CommandItem
                key={cmd.path}
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
        ))}
      </CommandList>
      <div className="sticky bg-background flex items-center space-x-1 mx-2 py-2 border-t">
        <kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
          <span className="text-xs">↑</span>
        </kbd>
        <kbd className="pointer-events-none inline-flex h-5 select-none items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
          <span className="text-xs">↓</span>
        </kbd>
        <p className="text-xs pr-2">to navigate</p>
        <kbd className="pointer-events-none inline-flex h-5 select-none items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
          <span className="text-xs">↵</span>
        </kbd>
        <p className="text-xs pr-2">to select</p>
        <kbd className="pointer-events-none inline-flex h-5 select-none items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
          <span className="text-xs">esc</span>
        </kbd>
        <p className="text-xs">to close</p>
      </div>
    </CommandDialog>
  );
}
