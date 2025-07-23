/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { SidebarGroup, SidebarGroupContent } from "@/components/ui/sidebar";
import { commandRoutes, quickActions } from "@/config/site-search";
import { SITE_SEARCH_RECENT_SEARCHES_KEY } from "@/constants/env";
import { useRecentSearches, useSearch } from "@/hooks/use-search";
import { cn } from "@/lib/utils";
import { SearchEntityType, SearchResult } from "@/types/search";
import { faHistory, faXmark } from "@fortawesome/pro-regular-svg-icons";
import * as React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Icon } from "../ui/icons";
import { VisuallyHidden } from "../ui/visually-hidden";
import { SiteSearchEmpty } from "./site-search-empty";
import { SiteSearchFooter } from "./site-search-footer";
import { SearchInputWithBadges, SiteSearchInput } from "./site-search-input";
import { SiteSearchLoading } from "./site-search-loading";
import {
  getResultComponent,
  SiteSearchQuickOption,
} from "./site-search-type-components";

export function SearchForm({ ...props }: React.ComponentProps<"form">) {
  return (
    <form {...props}>
      <SidebarGroup className="py-0">
        <SidebarGroupContent className="relative">
          <SiteSearchDialog />
        </SidebarGroupContent>
      </SidebarGroup>
    </form>
  );
}

export function SiteSearchDialog() {
  const [open, setOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const [activeFilters, setActiveFilters] = useState<Record<string, string>>(
    {},
  );
  const resultsRef = useRef<HTMLDivElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Use our custom hooks
  const {
    query: searchQuery,
    setQuery: setSearchQuery,
    results: searchResults,
    isLoading,
    activeTab,
    setActiveTab,
    refetch,
  } = useSearch();

  const { recentSearches, addRecentSearch, removeRecentSearch } =
    useRecentSearches(SITE_SEARCH_RECENT_SEARCHES_KEY, 5);

  const navigate = useNavigate();

  // Reset highlighted index when search results or dialog state changes
  useEffect(() => {
    setHighlightedIndex(0);
  }, [searchResults, open]);

  // Focus input when dialog opens
  useEffect(() => {
    if (open && inputRef.current) {
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  }, [open]);

  // Explicitly trigger search when dialog opens if there's a query
  useEffect(() => {
    if (open && searchQuery) {
      // Force a refetch when the dialog opens if there's an existing query
      refetch();
    }
  }, [open, searchQuery, refetch]);

  // Update search when filters change
  useEffect(() => {
    if (searchQuery) {
      refetch();
    }
  }, [activeFilters, refetch, searchQuery]);

  const handleNavigate = useCallback(
    (link: string) => {
      setOpen(false);
      navigate(link);
      setSearchQuery("");
      setActiveFilters({});
    },
    [navigate, setSearchQuery],
  );

  const handleResultClick = useCallback(
    (result: SearchResult) => {
      // Save search to recent
      addRecentSearch(searchQuery);

      // Determine where to navigate based on result type
      let link = "";

      switch (result.type) {
        case SearchEntityType.Shipments:
          link = `/shipments/management?entityId=${result.id}&modalType=edit`;
          break;
        case SearchEntityType.Workers:
          link = `/dispatch/configurations/workers?entityId=${result.id}&modalType=edit`;
          break;
        case SearchEntityType.Tractors:
          link = `/equipment/configurations/tractors?entityId=${result.id}&modalType=edit`;
          break;
        case SearchEntityType.Customers:
          link = `/billing/configurations/customers?entityId=${result.id}&modalType=edit`;
          break;
        default:
          // Default fallback if type is unknown
          link = `/search?q=${searchQuery}`;
      }

      setOpen(false);
      navigate(link);
      setSearchQuery("");
      setActiveFilters({});
    },
    [navigate, searchQuery, addRecentSearch, setSearchQuery],
  );

  // Handle keyboard shortcut
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!open) return;

      // Count all navigable items
      const recentLength = recentSearches.length;
      const resultsLength = searchResults?.length || 0;

      // Calculate total items to navigate
      const totalItems = searchQuery ? resultsLength : recentLength;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        setHighlightedIndex((prev) => {
          const newIndex = Math.min(prev + 1, totalItems - 1);
          scrollToHighlighted(newIndex);
          return newIndex;
        });
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setHighlightedIndex((prev) => {
          const newIndex = Math.max(prev - 1, 0);
          scrollToHighlighted(newIndex);
          return newIndex;
        });
      } else if (e.key === "Enter" && highlightedIndex >= 0) {
        e.preventDefault();

        if (searchQuery) {
          // Handle search results
          if (searchResults && searchResults[highlightedIndex]) {
            handleResultClick(searchResults[highlightedIndex]);
          }
        } else {
          // Handle recent searches
          if (recentSearches[highlightedIndex]) {
            setSearchQuery(recentSearches[highlightedIndex]);
          }
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [
    open,
    searchResults,
    recentSearches,
    searchQuery,
    highlightedIndex,
    handleResultClick,
    setSearchQuery,
  ]);

  const scrollToHighlighted = (index: number) => {
    if (resultsRef.current) {
      const elements = resultsRef.current.querySelectorAll(
        '[data-highlighted="true"]',
      );
      if (elements[index]) {
        elements[index].scrollIntoView({
          behavior: "smooth",
          block: "nearest",
        });
      }
    }
  };

  // Apply filters to the search results
  const filteredResults = React.useMemo(() => {
    if (!searchResults?.length) return [];

    // If no filters are active, return all results
    if (Object.keys(activeFilters).length === 0) {
      return searchResults;
    }

    // In a real implementation, you would filter the results based on activeFilters
    // For now, we'll just mock this by returning the same results
    // In a production environment, you would probably send the filters to the backend

    return searchResults;
  }, [searchResults, activeFilters]);

  // Group results by type for better organization
  const groupedResults = React.useMemo(() => {
    if (!filteredResults?.length) return {};

    return filteredResults.reduce<Record<string, SearchResult[]>>(
      (groups, result) => {
        const type = result.type;
        if (!groups[type]) {
          groups[type] = [];
        }
        groups[type].push(result);
        return groups;
      },
      {},
    );
  }, [filteredResults]);

  // Render recent searches
  const renderRecentSearches = () => {
    if (!recentSearches.length || searchQuery) return null;

    return (
      <div className="mb-4">
        <h3 className="text-sm font-medium text-muted-foreground mb-2">
          Recent Searches
        </h3>
        <div className="flex flex-col gap-1">
          {recentSearches.map((search, index) => (
            <div
              key={`recent-${index}`}
              className={cn(
                "flex items-center justify-between p-2 cursor-pointer rounded-md",
                highlightedIndex === index ? "bg-muted" : "hover:bg-muted",
              )}
              onClick={() => setSearchQuery(search)}
              data-highlighted={highlightedIndex === index}
            >
              <div className="flex items-center">
                <Icon
                  icon={faHistory}
                  className="mr-2 size-4 text-muted-foreground"
                />
                <span className="text-sm">{search}</span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="size-6 p-0 opacity-50 hover:opacity-100"
                onClick={(e) => {
                  e.stopPropagation();
                  removeRecentSearch(search);
                }}
              >
                <span className="sr-only">Remove</span>
                <Icon icon={faXmark} className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </div>
      </div>
    );
  };

  // Render search results
  const renderSearchResults = () => {
    if (!searchQuery) return null;

    if (isLoading) return <SiteSearchLoading />;

    if (!filteredResults || filteredResults.length === 0) {
      return <SiteSearchEmpty searchQuery={searchQuery} />;
    }

    // If we're showing all results without grouping
    if (activeTab === "all") {
      return (
        <div>
          <h3 className="text-sm font-medium text-muted-foreground mb-2">
            Search Results
          </h3>
          <div className="flex flex-col gap-1">
            {filteredResults.map((result, index) =>
              React.cloneElement(
                getResultComponent(result, {
                  searchQuery,
                  onClick: handleResultClick,
                  highlighted: highlightedIndex === index,
                }),
                { key: `result-${result.id}-${index}` },
              ),
            )}
          </div>
        </div>
      );
    }

    // Group by entity type
    return (
      <div className="flex flex-col gap-6">
        {Object.entries(groupedResults).map(([type, results]) => (
          <div key={type} className="flex flex-col gap-1">
            <h3 className="text-sm font-medium text-muted-foreground mb-2 capitalize">
              {type}s
            </h3>
            {results.map((result, index) => {
              const globalIndex = filteredResults.findIndex(
                (r) => r.id === result.id,
              );
              return React.cloneElement(
                getResultComponent(result, {
                  searchQuery,
                  onClick: handleResultClick,
                  highlighted: highlightedIndex === globalIndex,
                }),
                { key: `result-${result.id}-${index}` },
              );
            })}
          </div>
        ))}
      </div>
    );
  };

  const renderQuickActions = () => {
    // If there is a search query, don't show quick actions
    if (searchQuery) return null;

    return (
      <div className="flex flex-col gap-1">
        <h3 className="text-xs font-medium text-muted-foreground">
          Quick Actions
        </h3>
        <div className="flex flex-col gap-1">
          {Object.entries(quickActions).map(([key, action]) => (
            <SiteSearchQuickOption
              key={key}
              {...action}
              onClick={() => handleNavigate(action.link || "")}
            />
          ))}
        </div>
      </div>
    );
  };

  const renderApplicationRoutes = () => {
    if (searchQuery) return null;

    return (
      <div className="flex flex-col gap-1">
        <h3 className="text-xs font-medium text-muted-foreground">
          Navigation
        </h3>
        <div className="flex flex-col gap-1">
          {commandRoutes.map((group) => {
            return (
              <React.Fragment key={group.id}>
                <h4 className="text-xs font-medium text-muted-foreground">
                  {group.label}
                </h4>
                <div className="flex flex-col gap-1">
                  {group.routes.map((route) => (
                    <button
                      key={route.id}
                      className="flex items-center gap-2 p-2 rounded-md hover:bg-muted transition-colors cursor-pointer outline-none"
                      onClick={() => handleNavigate(route.link)}
                    >
                      {route.icon && (
                        <Icon
                          icon={route.icon}
                          className="size-4 text-muted-foreground"
                        />
                      )}
                      <span className="text-sm font-medium">{route.label}</span>
                    </button>
                  ))}
                </div>
              </React.Fragment>
            );
          })}
        </div>
      </div>
    );
  };

  return (
    <>
      <SiteSearchInput open={open} setOpen={setOpen} />
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          className="overflow-hidden p-0 sm:max-w-[650px]"
          ref={dialogRef}
          withClose={false}
        >
          <VisuallyHidden>
            <DialogHeader>
              <DialogTitle>Search</DialogTitle>
              <DialogDescription>
                Search for anything in your organization
              </DialogDescription>
            </DialogHeader>
          </VisuallyHidden>

          <SearchInputWithBadges
            searchQuery={searchQuery}
            setSearchQuery={setSearchQuery}
            activeTab={activeTab}
            setActiveTab={setActiveTab}
            inputRef={inputRef}
            activeFilters={activeFilters}
            setActiveFilters={setActiveFilters}
          />

          <div
            className="max-h-[60vh] overflow-auto py-4 px-3"
            ref={resultsRef}
          >
            {renderRecentSearches()}
            {renderApplicationRoutes()}
            {renderSearchResults()}
            {renderQuickActions()}
          </div>

          <SiteSearchFooter />
        </DialogContent>
      </Dialog>
    </>
  );
}
