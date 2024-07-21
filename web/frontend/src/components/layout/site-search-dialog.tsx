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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useTheme } from "@/components/ui/theme-provider";
import { useUserPermissions } from "@/context/user-permissions";
import { useUserFavorites } from "@/hooks/useQueries";
import { RECENT_SEARCH_KEY } from "@/lib/constants";
import { upperFirst } from "@/lib/utils";
import { RouteObjectWithPermission, routes } from "@/routing/AppRoutes";
import { useHeaderStore } from "@/stores/HeaderStore";
import { ThemeOptions } from "@/types";
import {
  faRectangleHistory,
  faSearch,
  faStarHalf,
} from "@fortawesome/pro-duotone-svg-icons";
import { faPage } from "@fortawesome/pro-solid-svg-icons";
import { LaptopIcon, MoonIcon, SunIcon } from "@radix-ui/react-icons";
import { VariantProps } from "class-variance-authority";
import debounce from "lodash-es/debounce";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { VisuallyHidden } from "react-aria";
import { useNavigate } from "react-router-dom";
import { Icon } from "../common/icons";

type BadgeVariant = VariantProps<typeof badgeVariants>["variant"];

function prepareRouteGroups(routeList: typeof routes) {
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
}

function Highlight({ text, highlight }: { text: string; highlight: string }) {
  if (!highlight.trim()) {
    return <span>{text}</span>;
  }
  const regex = new RegExp(`(${highlight})`, "gi");
  const parts = text.split(regex);
  return (
    <span>
      {parts.map((part, i) =>
        regex.test(part) ? (
          <mark key={i} className="bg-blue-300">
            {part}
          </mark>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
    </span>
  );
}

function SearchInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="flex items-center border-b px-3">
      <Icon icon={faSearch} className="text-muted-foreground mr-2" />
      <input
        placeholder="What do you need?"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="placeholder:text-muted-foreground flex h-11 w-full rounded-md bg-transparent py-3 text-sm outline-none disabled:cursor-not-allowed disabled:opacity-50"
      />
    </div>
  );
}

function RecentSearches({
  searches,
  onSelect,
  selectedIndex,
  startIndex,
  selectedItemRef,
}: {
  searches: string[];
  onSelect: (search: string) => void;
  selectedIndex: number;
  startIndex: number;
  selectedItemRef: React.RefObject<HTMLDivElement>;
}) {
  return (
    <div className="overflow-hidden p-1">
      <div className="text-muted-foreground px-1 py-1.5 text-xs font-medium">
        Recent Searches
      </div>
      {searches.map((search, index) => {
        const itemIndex = startIndex + index;
        const isSelected = selectedIndex === itemIndex;
        return (
          <div
            key={search}
            ref={isSelected ? selectedItemRef : null}
            className={`hover:bg-muted relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none ${
              isSelected ? "bg-muted" : ""
            }`}
            onClick={() => onSelect(search)}
            data-index={itemIndex}
          >
            <div className="flex items-center justify-center">
              <div className="mr-2 flex size-8 items-center justify-center rounded-md border border-purple-200 bg-purple-200 text-purple-600 dark:border-purple-500 dark:bg-purple-600/30 dark:text-purple-400 forced-colors:outline">
                <Icon icon={faRectangleHistory} className="size-4" />
              </div>
            </div>
            {search}
          </div>
        );
      })}
    </div>
  );
}

function FavoriteRoutes({
  routes,
  inputValue,
  onNavigate,
  mapGroupVariant,
  selectedIndex,
  startIndex,
  selectedItemRef,
}: {
  routes: RouteObjectWithPermission[];
  inputValue: string;
  onNavigate: (path: string) => void;
  mapGroupVariant: (group: string) => BadgeVariant;
  selectedIndex: number;
  startIndex: number;
  selectedItemRef: React.RefObject<HTMLDivElement>;
}) {
  return (
    <div className="overflow-hidden p-1">
      <div className="text-muted-foreground px-1 py-1.5 text-xs font-medium">
        Favorites
      </div>
      {routes.map((route, index) => {
        const itemIndex = startIndex + index;
        const isSelected = selectedIndex === itemIndex;

        return (
          <div
            key={route.path}
            ref={isSelected ? selectedItemRef : null}
            className={`hover:bg-muted relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none ${
              isSelected ? "bg-muted" : ""
            }`}
            onClick={() => onNavigate(route.path as string)}
            data-index={itemIndex}
          >
            <div className="flex items-center justify-center">
              <div className="forced-colors:outlin mr-2 flex size-8 items-center justify-center rounded-md border border-yellow-200 bg-yellow-200 text-yellow-600 dark:border-yellow-500 dark:bg-yellow-600/30 dark:text-yellow-400">
                <Icon icon={faStarHalf} className="size-4" />
              </div>
            </div>
            <Highlight text={route.title} highlight={inputValue} />
            <Badge variant={mapGroupVariant(route.group)} className="ml-auto">
              {upperFirst(route.group)}
            </Badge>
          </div>
        );
      })}
    </div>
  );
}

function FilteredGroups({
  groups,
  inputValue,
  onNavigate,
  selectedIndex,
  startIndex,
  selectedItemRef,
}: {
  groups: Record<string, RouteObjectWithPermission[]>;
  inputValue: string;
  onNavigate: (path: string) => void;
  selectedIndex: number;
  startIndex: number;
  selectedItemRef: React.RefObject<HTMLDivElement>;
}) {
  let currentIndex = startIndex;
  return (
    <>
      {Object.entries(groups).map(([group, groupCommands]) => (
        <div
          key={group}
          className="overflow-hidden p-1 text-zinc-950 dark:text-zinc-50"
        >
          <div className="px-1 py-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
            {upperFirst(group)}
          </div>
          {groupCommands.map((cmd) => {
            const isSelected = selectedIndex === currentIndex;
            currentIndex++;
            return (
              <div
                ref={isSelected ? selectedItemRef : null}
                key={cmd.path}
                className={`hover:bg-muted relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none ${
                  isSelected ? "bg-muted" : ""
                }`}
                onClick={() => onNavigate(cmd.path as string)}
                data-index={currentIndex - 1}
              >
                <div className="flex items-center justify-center">
                  <div className="mr-2 flex size-8 items-center justify-center rounded-md border border-blue-200 bg-blue-200 text-blue-600 dark:border-blue-500 dark:bg-blue-600/30 dark:text-blue-400 forced-colors:outline">
                    <Icon icon={faPage} className="size-4" />
                  </div>
                </div>
                <Highlight text={cmd.title} highlight={inputValue} />
              </div>
            );
          })}
        </div>
      ))}
    </>
  );
}

function NoResults({
  inputValue,
  popularSearches,
  onPopularSearch,
}: {
  inputValue: string;
  popularSearches: string[];
  onPopularSearch: (search: string) => void;
}) {
  return (
    <div className="p-4 text-center">
      <div className="text-sm font-medium">
        No results found for "{inputValue}"
      </div>
      <div className="text-muted-foreground mt-2 text-sm">
        Try one of these popular searches:
      </div>
      <div className="mt-4 flex flex-wrap justify-center gap-2">
        {popularSearches.map((search) => (
          <Badge
            key={search}
            variant="outline"
            className="cursor-pointer"
            onClick={() => onPopularSearch(search)}
          >
            {search}
          </Badge>
        ))}
      </div>
    </div>
  );
}

function SearchFooter({
  setTheme,
}: {
  setTheme: (theme: ThemeOptions) => void;
}) {
  return (
    <div className="flex items-center justify-between border-t px-3 py-2 text-sm">
      <div className="text-muted-foreground flex space-x-1 text-xs">
        <span>&#8593;</span>
        <span>&#8595;</span>
        <p className="pr-2">to navigate</p>
        <span>&#x23CE;</span>
        <p className="pr-2">to select</p>
        <span>esc</span>
        <p>to close</p>
      </div>
      <div className="flex space-x-2">
        <SunIcon
          className="size-4 cursor-pointer hover:text-zinc-900 dark:hover:text-zinc-50"
          onClick={() => setTheme("light")}
        />
        <MoonIcon
          className="size-4 cursor-pointer hover:text-zinc-900 dark:hover:text-zinc-50"
          onClick={() => setTheme("dark")}
        />
        <LaptopIcon
          className="size-4 cursor-pointer hover:text-zinc-900 dark:hover:text-zinc-50"
          onClick={() => setTheme("system")}
        />
      </div>
    </div>
  );
}

export function SiteSearchDialog() {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const [open, setOpen] = useHeaderStore.use("searchDialogOpen");
  const { data: userFavorites } = useUserFavorites();
  const [inputValue, setInputValue] = useState("");
  const [recentSearches, setRecentSearches] = useState<string[]>([]);
  const [filteredGroups, setFilteredGroups] = useState<
    Record<string, RouteObjectWithPermission[]>
  >({});
  const { setTheme } = useTheme();

  const [selectedIndex, setSelectedIndex] = useState(-1);
  const dialogRef = useRef<HTMLDivElement>(null);
  const selectedItemRef = useRef<HTMLDivElement>(null);

  // Load recent searches from local storage on component mount
  useEffect(() => {
    const storedSearches = localStorage.getItem(RECENT_SEARCH_KEY);
    if (storedSearches) {
      setRecentSearches(JSON.parse(storedSearches));
    }
  }, []);

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

  const favoritePaths = useMemo(
    () =>
      new Set((userFavorites as any[])?.map((favorite) => favorite.pageLink)),
    [userFavorites],
  );

  const popularSearches = [
    "Dashboard",
    "Shipments",
    "Invoices",
    "Reports",
    "Settings",
  ];

  const filterAndHighlight = useCallback(
    (value: string) => {
      const filtered = routes.filter((route) => {
        if (route.excludeFromMenu) return false;
        if (route.path.endsWith("/:id")) return false;
        if (route.permission && !userHasPermission(route.permission))
          return false;
        return (
          isAuthenticated &&
          route.title.toLowerCase().includes(value.toLowerCase())
        );
      });
      const grouped = prepareRouteGroups(filtered);
      setFilteredGroups(grouped);
    },
    [isAuthenticated, userHasPermission],
  );

  const debouncedFilter = useMemo(
    () => debounce(filterAndHighlight, 300),
    [filterAndHighlight],
  );

  useEffect(() => {
    debouncedFilter(inputValue);
    return () => debouncedFilter.cancel();
  }, [inputValue, debouncedFilter]);

  function handleSearchChange(value: string) {
    setInputValue(value);
  }

  function handleDialogOpenChange(isOpen: boolean) {
    setOpen(isOpen);
    if (!isOpen) {
      setInputValue("");
    }
  }

  const groupVariants: Record<string, BadgeVariant>[] = [
    { administration: "warning" },
    { main: "default" },
    { billing: "inactive" },
    { accounting: "pink" },
    { dispatch: "info" },
    { equipment: "purple" },
  ];

  function mapGroupVariant(group: string): BadgeVariant {
    const variant = groupVariants.find((v) => v[group]);
    return variant ? variant[group] : "outline";
  }

  const favoriteRoutes = useMemo(
    () =>
      Object.values(filteredGroups)
        .flat()
        .filter((route) => favoritePaths.has(route.path)),
    [filteredGroups, favoritePaths],
  );

  function handleNavigate(path: string, title: string) {
    navigate(path);
    setOpen(false);

    // Update recent searches
    const updatedSearches = [
      title,
      ...recentSearches.filter((s) => s !== title),
    ].slice(0, 5);
    setRecentSearches(updatedSearches);
    localStorage.setItem(RECENT_SEARCH_KEY, JSON.stringify(updatedSearches));
  }

  const allItems = useMemo(() => {
    return [
      ...recentSearches.map((search) => ({ type: "recent", value: search })),
      ...favoriteRoutes.map((route) => ({ type: "favorite", value: route })),
      ...Object.values(filteredGroups)
        .flat()
        .map((route) => ({ type: "route", value: route })),
    ];
  }, [recentSearches, favoriteRoutes, filteredGroups]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prevIndex) => {
            const nextIndex = prevIndex + 1;
            return nextIndex < allItems.length ? nextIndex : 0;
          });
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prevIndex) => {
            const nextIndex = prevIndex - 1;
            return nextIndex >= 0 ? nextIndex : allItems.length - 1;
          });
          break;
        case "Enter":
          e.preventDefault();
          if (selectedIndex >= 0 && selectedIndex < allItems.length) {
            const selectedItem = allItems[selectedIndex];
            if (selectedItem.type === "recent") {
              const route = routes.find((r) => r.title === selectedItem.value);
              if (route) handleNavigate(route.path as string, route.title);
            } else if (
              selectedItem.type === "favorite" ||
              selectedItem.type === "route"
            ) {
              if (typeof selectedItem.value !== "string") {
                handleNavigate(
                  selectedItem.value.path as string,
                  selectedItem.value.title,
                );
              }
            }
          }
          break;
        case "Escape":
          e.preventDefault();
          setOpen(false);
          break;
      }
    },
    [selectedIndex, allItems, handleNavigate, setOpen, routes],
  );

  useEffect(() => {
    setSelectedIndex(-1);
  }, [inputValue, allItems.length]);

  useEffect(() => {
    if (selectedIndex >= 0 && selectedItemRef.current) {
      selectedItemRef.current.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
      });
    }
  }, [selectedIndex]);

  const renderSearchResults = () => {
    let currentIndex = 0;

    return (
      <>
        {inputValue === "" && recentSearches.length > 0 && (
          <RecentSearches
            searches={recentSearches}
            onSelect={(search) => {
              const route = routes.find((r) => r.title === search);
              if (route) handleNavigate(route.path as string, route.title);
            }}
            selectedIndex={selectedIndex}
            startIndex={currentIndex}
            selectedItemRef={selectedItemRef}
          />
        )}
        {(() => {
          currentIndex += recentSearches.length;
          return null;
        })()}
        {favoriteRoutes.length > 0 && (
          <FavoriteRoutes
            routes={favoriteRoutes}
            inputValue={inputValue}
            onNavigate={(path) => {
              const route = routes.find((r) => r.path === path);
              if (route) handleNavigate(path, route.title);
            }}
            mapGroupVariant={mapGroupVariant}
            selectedIndex={selectedIndex}
            startIndex={currentIndex}
            selectedItemRef={selectedItemRef}
          />
        )}
        {(() => {
          currentIndex += favoriteRoutes.length;
          return null;
        })()}
        {Object.entries(filteredGroups).length > 0 ? (
          <FilteredGroups
            groups={filteredGroups}
            inputValue={inputValue}
            onNavigate={(path) => {
              const route = routes.find((r) => r.path === path);
              if (route) handleNavigate(path, route.title);
            }}
            selectedIndex={selectedIndex}
            startIndex={currentIndex}
            selectedItemRef={selectedItemRef}
          />
        ) : inputValue !== "" ? (
          <NoResults
            inputValue={inputValue}
            popularSearches={popularSearches}
            onPopularSearch={handleSearchChange}
          />
        ) : null}
      </>
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent
        className="bg-card/95 supports-[backdrop-filter]:bg-background/40 overflow-hidden p-0 shadow-lg backdrop-blur sm:max-w-[600px]"
        ref={dialogRef}
        onKeyDown={handleKeyDown}
      >
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Site Search</DialogTitle>
            <DialogDescription>
              Search for pages, reports, and more
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <div className="flex size-full flex-col overflow-hidden rounded-md text-zinc-950 dark:text-zinc-50">
          <SearchInput value={inputValue} onChange={handleSearchChange} />
          <div className="max-h-[300px] overflow-y-auto overflow-x-hidden">
            {renderSearchResults()}
          </div>
          <SearchFooter setTheme={setTheme} />
        </div>
      </DialogContent>
    </Dialog>
  );
}
