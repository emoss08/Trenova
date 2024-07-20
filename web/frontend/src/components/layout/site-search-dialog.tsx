/**
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

import { Badge, badgeVariants } from "@/components/ui/badge";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { useTheme } from "@/components/ui/theme-provider";
import { useUserPermissions } from "@/context/user-permissions";
import { useUserFavorites } from "@/hooks/useQueries";
import { upperFirst } from "@/lib/utils";
import { RouteObjectWithPermission, routes } from "@/routing/AppRoutes";
import { useHeaderStore } from "@/stores/HeaderStore";
import { faPage } from "@fortawesome/pro-regular-svg-icons";
import { faHistory, faSearch, faStar } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  LaptopIcon,
  MoonIcon,
  SunIcon
} from "@radix-ui/react-icons";
import { VariantProps } from "class-variance-authority";
import debounce from "lodash-es/debounce";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";


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
  

type BadgeVariant = VariantProps<typeof badgeVariants>['variant'];

// Highlight component to wrap matched text
const Highlight = ({ text, highlight }: { text: string; highlight: string }) => {
  if (!highlight.trim()) {
    return <span>{text}</span>;
  }
  const regex = new RegExp(`(${highlight})`, 'gi');
  const parts = text.split(regex);
  return (
    <span>
      {parts.map((part, i) => 
        regex.test(part) ? <mark key={i} className="bg-blue-800 dark:bg-blue-300">{part}</mark> : <span key={i}>{part}</span>
      )}
    </span>
  );
};

export function SiteSearchDialog() {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const [open, setOpen] = useHeaderStore.use('searchDialogOpen');
  const { data: userFavorites } = useUserFavorites();
  const [inputValue, setInputValue] = useState('');
  const [recentSearches, setRecentSearches] = useState<string[]>([]);
  const [filteredGroups, setFilteredGroups] = useState<Record<string, RouteObjectWithPermission[]>>({});
  const { setTheme } = useTheme();
  const [selectedIndex, _] = useState(-1);
  
  useEffect(() => {
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

  const favoritePaths = useMemo(() => new Set(
    (userFavorites as any[])?.map((favorite) => favorite.pageLink)
  ), [userFavorites]);

  const popularSearches = ['Dashboard', 'Shipments', 'Invoices', 'Reports', 'Settings'];

  const debouncedFilter = useCallback(
    debounce((value: string) => {
      const filtered = routes.filter((route) => {
        if (route.excludeFromMenu) return false;
        if (route.path.endsWith('/:id')) return false;
        if (route.permission && !userHasPermission(route.permission)) return false;
        return isAuthenticated && route.title.toLowerCase().includes(value.toLowerCase());
      });
      const grouped = prepareRouteGroups(filtered);
      setFilteredGroups(grouped);
    }, 300),
    [isAuthenticated, userHasPermission]
  );

  useEffect(() => {
    debouncedFilter(inputValue);
  }, [inputValue, debouncedFilter]);

  const handleSearchChange = (value: string) => {
    setInputValue(value);
    if (value.trim() !== '') {
      setRecentSearches((prev) => [value, ...prev.filter((s) => s !== value)].slice(0, 5));
    }
  };

  const handleDialogOpenChange = (isOpen: boolean) => {
    setOpen(isOpen);
    if (!isOpen) {
      setInputValue('');
    }
  };

  const groupVariants: Record<string, BadgeVariant>[] = [
    { administration: 'warning' },
    { main: 'default' },
    { billing: 'inactive' },
    { accounting: 'pink' },
    { dispatch: 'info' },
    { equipment: 'purple' },
  ];

  const mapGroupVariant = (group: string): BadgeVariant => {
    const variant = groupVariants.find((v) => v[group]);
    return variant ? variant[group] : 'outline';
  };

  const upperFirstGroup = (group: string) => upperFirst(group);

  const favoriteRoutes = useMemo(() => 
    Object.values(filteredGroups).flat().filter(route => favoritePaths.has(route.path)),
    [filteredGroups, favoritePaths]
  );

  const handleNavigate = (path: string) => {
    navigate(path);
    setOpen(false);
  };

  const filterAndHighlight = useCallback((value: string) => {
    const filtered = routes.filter((route) => {
      if (route.excludeFromMenu) return false;
      if (route.path.endsWith('/:id')) return false;
      if (route.permission && !userHasPermission(route.permission)) return false;
      return isAuthenticated && route.title.toLowerCase().includes(value.toLowerCase());
    });
    const grouped = prepareRouteGroups(filtered);
    setFilteredGroups(grouped);
  }, [isAuthenticated, userHasPermission]);

  useEffect(() => {
    filterAndHighlight(inputValue);
  }, [inputValue, filterAndHighlight]);

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent className="overflow-hidden p-0 shadow-lg">
        <div className="flex h-full w-full flex-col overflow-hidden rounded-md bg-white text-zinc-950 dark:bg-zinc-950 dark:text-zinc-50">
          <div className="flex items-center border-b px-3">
            <FontAwesomeIcon icon={faSearch} className="mr-2 text-muted-foreground" />
            <input
              placeholder="What do you need?"
              value={inputValue}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="flex h-11 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50"
            />
          </div>
          <div className="max-h-[300px] overflow-y-auto overflow-x-hidden">
            {inputValue === '' && recentSearches.length > 0 && (
              <div className="overflow-hidden p-1 text-zinc-950 dark:text-zinc-50">
                <div className="px-1 py-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  Recent Searches
                </div>
                {recentSearches.map((search, index) => (
                  <div
                    key={search}
                    className={`relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none ${
                      index === selectedIndex ? 'bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-50' : 'hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-50'
                    }`}
                    onClick={() => handleSearchChange(search)}
                  >
                    <FontAwesomeIcon icon={faHistory} className="mr-2 size-4" />
                    {search}
                  </div>
                ))}
              </div>
            )}
            {favoriteRoutes.length > 0 && (
              <div className="overflow-hidden p-1 text-zinc-950 dark:text-zinc-50">
                <div className="px-1 py-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  Favorites
                </div>
                {favoriteRoutes.map((route, index) => (
                  <div
                    key={route.path}
                    className={`relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none ${
                      recentSearches.length + index === selectedIndex ? 'bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-50' : 'hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-50'
                    }`}
                    onClick={() => handleNavigate(route.path as string)}
                  >
                    <FontAwesomeIcon icon={faStar} className="mr-2 size-4" />
                    <Highlight text={route.title} highlight={inputValue} />
                    <Badge variant={mapGroupVariant(route.group)} className="ml-auto">
                      {upperFirstGroup(route.group)}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
            {Object.entries(filteredGroups).length > 0 ? (
              Object.entries(filteredGroups).map(([group, groupCommands]) => (
                <div key={group} className="overflow-hidden p-1 text-zinc-950 dark:text-zinc-50">
                  <div className="px-1 py-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                    {upperFirst(group)}
                  </div>
                  {groupCommands.map((cmd, index) => (
                    <div
                      key={cmd.path}
                      className={`relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none ${
                        recentSearches.length + favoriteRoutes.length + index === selectedIndex ? 'bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-50' : 'hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-50'
                      }`}
                      onClick={() => handleNavigate(cmd.path as string)}
                    >
                      <FontAwesomeIcon icon={faPage} className="mr-2 size-4" />
                      <Highlight text={cmd.title} highlight={inputValue} />
                    </div>
                  ))}
                </div>
              ))
            ) : inputValue !== '' ? (
              <div className="p-4 text-center">
                <div className="text-sm font-medium">No results found for "{inputValue}"</div>
                <div className="mt-2 text-sm text-muted-foreground">
                  Try one of these popular searches:
                </div>
                <div className="mt-4 flex flex-wrap justify-center gap-2">
                  {popularSearches.map((search) => (
                    <Badge
                      key={search}
                      variant="outline"
                      className="cursor-pointer"
                      onClick={() => handleSearchChange(search)}
                    >
                      {search}
                    </Badge>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
          <div className="flex items-center justify-between border-t px-3 py-2 text-sm">
            <div className="flex space-x-1 text-xs text-muted-foreground">
              <span>&#8593;</span>
              <span>&#8595;</span>
              <p className="pr-2">to navigate</p>
              <span>&#x23CE;</span>
              <p className="pr-2">to select</p>
              <span>esc</span>
              <p>to close</p>
            </div>
            <div className="flex space-x-2">
              <SunIcon className="cursor-pointer size-4 hover:text-zinc-900 dark:hover:text-zinc-50" onClick={() => setTheme("light")} />
              <MoonIcon className="cursor-pointer size-4 hover:text-zinc-900 dark:hover:text-zinc-50" onClick={() => setTheme("dark")} />
              <LaptopIcon className="cursor-pointer size-4 hover:text-zinc-900 dark:hover:text-zinc-50" onClick={() => setTheme("system")} />
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}