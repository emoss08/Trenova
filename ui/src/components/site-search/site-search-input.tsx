/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { tabConfig } from "@/config/site-search";
import { cn } from "@/lib/utils";
import { SearchInputProps, SiteSearchTab } from "@/types/search";
import {
  faChevronDown,
  faChevronUp,
  faCommand,
  faMagnifyingGlass,
  faSearch,
  faXmark,
} from "@fortawesome/pro-regular-svg-icons";
import { AnimatePresence, motion } from "motion/react";
import React, { useEffect, useRef, useState } from "react";
import { getFilterOptions } from "./site-search-filter-options";

export function SiteSearchInput({
  open,
  setOpen,
}: {
  open: boolean;
  setOpen: (open: boolean) => void;
}) {
  return (
    <span
      aria-label="Open site search"
      aria-expanded={open}
      title="Open site search"
      onClick={() => setOpen(true)}
      className="group hidden h-8 items-center justify-between rounded-md border border-muted-foreground/20 bg-background px-3 py-2 text-sm hover:cursor-pointer hover:border-muted-foreground/80 xl:flex"
    >
      <div className="flex items-center">
        <Icon
          icon={faMagnifyingGlass}
          className="mr-2 size-3.5 text-muted-foreground group-hover:text-foreground"
        />
        <span className="text-muted-foreground">Search...</span>
      </div>
      <div className="pointer-events-none inline-flex select-none">
        <kbd className="-me-1 ms-12 inline-flex h-5 max-h-full items-center rounded border border-border bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-muted-foreground/70">
          ⌘K
        </kbd>
      </div>
    </span>
  );
}

export function SearchInputWithBadges({
  searchQuery,
  setSearchQuery,
  activeTab,
  setActiveTab,
  inputRef,
  activeFilters = {},
  setActiveFilters = () => {},
}: SearchInputProps) {
  const [showFilters, setShowFilters] = useState(false);
  const [showTagSuggestions, setShowTagSuggestions] = useState(false);
  const [showFilterDropdown, setShowFilterDropdown] = useState(false);
  const [tagFilter, setTagFilter] = useState("");
  const [selectedTagIndex, setSelectedTagIndex] = useState(0);
  const [cursorPosition, setCursorPosition] = useState(0);
  const [atSymbolIndex, setAtSymbolIndex] = useState(-1);
  const [previousActiveTab, setPreviousActiveTab] =
    useState<SiteSearchTab>(activeTab);
  const filtersContainerRef = useRef<HTMLDivElement>(null);
  const inputContainerRef = useRef<HTMLDivElement>(null);
  const moreButtonRef = useRef<HTMLButtonElement>(null);

  // Update filter visibility when tab changes
  useEffect(() => {
    setShowFilters(activeTab !== "all");

    // If the active tab has changed (not just on initial render), clear the filters
    if (previousActiveTab !== activeTab && previousActiveTab !== undefined) {
      setActiveFilters({});
    }

    // Update previous active tab
    setPreviousActiveTab(activeTab);
  }, [activeTab, previousActiveTab, setActiveFilters]);

  // Close dropdown when filters change
  useEffect(() => {
    setShowFilterDropdown(false);
  }, [activeFilters]);

  // Handle clicks outside the dropdown to close it
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        showFilterDropdown &&
        moreButtonRef.current &&
        !moreButtonRef.current.contains(event.target as Node)
      ) {
        setShowFilterDropdown(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showFilterDropdown]);

  // Handle tag suggestions visibility and filtering
  useEffect(() => {
    if (!searchQuery) {
      setShowTagSuggestions(false);
      return;
    }

    const atIndex = searchQuery.lastIndexOf("@");

    if (atIndex === -1 || atSymbolIndex !== atIndex) {
      // Reset tag suggestions if no @ symbol or if it's a new @ symbol
      setShowTagSuggestions(false);
      setTagFilter("");
      setAtSymbolIndex(-1);
      return;
    }

    // Check if there's a space after the @ symbol
    const hasSpaceAfterAt = searchQuery.substring(atIndex).includes(" ");
    if (hasSpaceAfterAt) {
      setShowTagSuggestions(false);
      return;
    }

    // Get the partial tag after the @ symbol
    const partialTag = searchQuery.substring(atIndex + 1);
    setTagFilter(partialTag.toLowerCase());
    setShowTagSuggestions(true);
  }, [searchQuery, atSymbolIndex]);

  // Filter tabs based on tag input
  const filteredTabs = Object.entries(tabConfig)
    .filter(
      ([key, config]) =>
        key !== "all" &&
        config.label.toLowerCase().includes(tagFilter.toLowerCase()),
    )
    .sort(([, configA], [, configB]) =>
      configA.label.toLowerCase().localeCompare(configB.label.toLowerCase()),
    );

  // Reset selected index when filtered tabs change
  useEffect(() => {
    setSelectedTagIndex(0);
  }, [filteredTabs.length]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setSearchQuery(newValue);

    // Track cursor position
    setCursorPosition(e.target.selectionStart ?? 0);

    // Check for @ symbol
    const atIndex = newValue.lastIndexOf("@");
    if (atIndex !== -1 && atIndex < cursorPosition) {
      // Only set if the @ is before the current cursor position
      setAtSymbolIndex(atIndex);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Handle Ctrl+Backspace to remove tag
    if (
      e.key === "Backspace" &&
      (e.ctrlKey || e.metaKey) &&
      activeTab !== "all"
    ) {
      e.preventDefault();
      setActiveTab("all");
      setShowFilters(false);
      // Clear filters when removing tag
      setActiveFilters({});
      return;
    }

    // Close dropdown on escape key
    if (e.key === "Escape" && showFilterDropdown) {
      e.preventDefault();
      setShowFilterDropdown(false);
      return;
    }

    if (!showTagSuggestions) return;

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setSelectedTagIndex((prev) =>
          prev < filteredTabs.length - 1 ? prev + 1 : prev,
        );
        break;
      case "ArrowUp":
        e.preventDefault();
        setSelectedTagIndex((prev) => (prev > 0 ? prev - 1 : 0));
        break;
      case "Enter":
        e.preventDefault();
        if (filteredTabs.length > 0) {
          applyTag(filteredTabs[selectedTagIndex][0]);
        }
        break;
      case "Escape":
        e.preventDefault();
        setShowTagSuggestions(false);
        break;
      case "Tab":
        e.preventDefault();
        if (filteredTabs.length > 0) {
          applyTag(filteredTabs[selectedTagIndex][0]);
        }
        break;
    }
  };

  const applyTag = (tabKey: string) => {
    // If changing to a different tag, clear filters first
    if (activeTab !== tabKey) {
      setActiveFilters({});
    }

    // Apply tag by setting active tab
    setActiveTab(tabKey as SiteSearchTab);
    setShowTagSuggestions(false);

    // Remove the @tag part from search query
    if (atSymbolIndex !== -1) {
      const beforeAt = searchQuery.substring(0, atSymbolIndex);
      const afterTag = searchQuery.substring(cursorPosition);
      setSearchQuery(beforeAt + afterTag);
    }

    // Reset state
    setAtSymbolIndex(-1);
    setTagFilter("");
  };

  const handleRemoveTab = (e: React.MouseEvent) => {
    e.stopPropagation();
    setActiveTab("all");
    setShowFilters(false);
    // Clear filters when changing tabs
    setActiveFilters({});
  };

  const handleAddFilter = (filter: string, value: string) => {
    setActiveFilters({
      ...activeFilters,
      [filter]: value,
    });
  };

  const handleRemoveFilter = (filter: string) => {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { [filter]: _, ...newFilters } = activeFilters;
    setActiveFilters(newFilters);
  };

  const toggleFilterDropdown = (e: React.MouseEvent) => {
    e.stopPropagation();
    setShowFilterDropdown(!showFilterDropdown);
  };

  // Get config for current tab, use the enhanced config
  const currentTabConfig = tabConfig[activeTab] || tabConfig.all;

  // Create array of filter entries for easier manipulation
  const filterEntries = Object.entries(activeFilters);
  const totalFilterCount = filterEntries.length;

  // Show only 1 visible filter when there are more than 1 filters
  const visibleFilters = filterEntries.slice(0, Math.min(1, totalFilterCount));
  const hiddenFilters = filterEntries.slice(1);
  const hiddenFilterCount = hiddenFilters.length;

  return (
    <>
      <div className="relative border-b border-border">
        <div className="pointer-events-none absolute inset-y-0 gap-1.5 left-0 flex items-center pl-2 text-xs text-muted-foreground">
          <Icon icon={faSearch} className="size-4" />
        </div>

        <div
          ref={inputContainerRef}
          className="flex items-center min-h-[48px] w-full px-8 py-2 relative"
        >
          <div className="flex items-center w-full">
            {/* Tab badge when not "all" */}
            <AnimatePresence>
              {activeTab !== "all" && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.8, y: -5 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.8, y: -5 }}
                  transition={{
                    type: "spring",
                    stiffness: 500,
                    damping: 30,
                  }}
                  className={cn(
                    "mr-1.5 text-xs flex items-center px-1.5 py-0.5 rounded-md flex-shrink-0",
                    currentTabConfig.color,
                  )}
                >
                  <span className="capitalize">{currentTabConfig.label}</span>
                  <button
                    onClick={handleRemoveTab}
                    className="ml-1 hover:bg-background/20 rounded-full size-4 inline-flex items-center justify-center cursor-pointer"
                  >
                    <Icon icon={faXmark} className="size-3" />
                  </button>
                </motion.div>
              )}
            </AnimatePresence>

            {/* Filter badges container - using flex layout with items-center to ensure vertical alignment */}
            <div
              ref={filtersContainerRef}
              className="flex items-center gap-1.5"
            >
              <AnimatePresence>
                {visibleFilters.map(([filter, filterValue]) => (
                  <motion.div
                    key={filter}
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.8 }}
                    transition={{
                      type: "spring",
                      stiffness: 500,
                      damping: 30,
                    }}
                    className="text-xs bg-muted text-accent-foreground flex items-center gap-1 px-1.5 py-0.5 rounded-md flex-shrink-0"
                  >
                    <span className="capitalize text-muted-foreground">
                      {filter}:
                    </span>
                    <span className="capitalize">
                      {filterValue.replace(/_/g, " ")}
                    </span>
                    <button
                      onClick={() => handleRemoveFilter(filter)}
                      className="ml-1 hover:bg-background/20 rounded-full size-4 inline-flex items-center justify-center"
                    >
                      <Icon icon={faXmark} className="size-3 cursor-pointer" />
                    </button>
                  </motion.div>
                ))}

                {/* "X more..." button */}
                {hiddenFilterCount > 0 && (
                  <motion.button
                    ref={moreButtonRef}
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.8 }}
                    onClick={toggleFilterDropdown}
                    className={cn(
                      "relative text-xs flex items-center gap-1 px-1.5 py-0.5 rounded-md cursor-pointer flex-shrink-0 mr-2.5",
                      showFilterDropdown
                        ? "bg-muted text-accent-foreground"
                        : "bg-muted text-muted-foreground hover:bg-muted",
                    )}
                  >
                    <span>{hiddenFilterCount} more...</span>
                    <Icon
                      icon={showFilterDropdown ? faChevronUp : faChevronDown}
                      className="size-3 ml-0.5"
                    />

                    {/* Filter dropdown */}
                    <AnimatePresence>
                      {showFilterDropdown && (
                        <motion.div
                          initial={{ opacity: 0, y: -5 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0, y: -5 }}
                          transition={{ duration: 0.15 }}
                          className="absolute left-0 top-full mt-1 z-50 min-w-[200px] max-w-[300px] bg-popover border border-border rounded-md shadow-md overflow-hidden cursor-auto"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <div className="p-1 max-h-[200px] overflow-y-auto">
                            <div className="px-2 py-1.5 text-xs text-muted-foreground font-medium border-b border-border mb-1 text-left">
                              Active Filters
                            </div>
                            {hiddenFilters.map(([filter, filterValue]) => (
                              <div
                                key={filter}
                                className="flex items-center justify-between px-2 py-1.5 text-sm hover:bg-muted rounded-sm cursor-auto"
                              >
                                <div className="flex items-center gap-1">
                                  <span className="capitalize text-xs text-muted-foreground">
                                    {filter}:
                                  </span>
                                  <span className="capitalize">
                                    {filterValue.replace(/_/g, " ")}
                                  </span>
                                </div>
                                <button
                                  onClick={() => handleRemoveFilter(filter)}
                                  className="ml-2 hover:bg-background/20 rounded-full size-5 inline-flex items-center justify-center"
                                >
                                  <Icon
                                    icon={faXmark}
                                    className="size-3 cursor-pointer"
                                  />
                                </button>
                              </div>
                            ))}
                          </div>
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </motion.button>
                )}
              </AnimatePresence>
            </div>

            {/* Input field - with added self-center for better vertical alignment */}
            <input
              ref={inputRef}
              placeholder={
                activeTab === "all"
                  ? "Search for anything or type @ for categories"
                  : `Search in ${currentTabConfig.label.toLowerCase()}`
              }
              value={searchQuery}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              className="flex h-full w-full min-w-[70px] bg-transparent text-sm outline-none placeholder:text-muted-foreground border-none self-center mr-4"
            />
          </div>

          <div className="pointer-events-none absolute inset-y-0 gap-1.5 right-0 flex items-center pr-3 text-xs text-muted-foreground">
            <kbd className="-me-1 inline-flex h-5 max-h-full items-center justify-center rounded bg-foreground/10 px-1 font-[inherit] font-medium text-foreground">
              <Icon icon={faCommand} className="size-3" />
            </kbd>
            <kbd className="-me-1 inline-flex h-5 text-xs max-h-full items-center justify-center rounded bg-foreground/10 px-1 font-[inherit] font-medium text-foreground">
              K
            </kbd>
          </div>
        </div>
      </div>

      {/* Tab suggestions dropdown */}
      <AnimatePresence>
        {showTagSuggestions && filteredTabs.length > 0 && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.15 }}
            className="absolute top-9 left-6 z-50 mt-1 w-64 bg-background border border-border rounded-md shadow-md max-h-60 overflow-y-auto"
          >
            <div className="p-1">
              <div className="px-2 py-1.5 text-xs text-muted-foreground font-medium border-b border-border mb-1">
                Select a category
              </div>
              {filteredTabs.map(([key, config], index) => (
                <div
                  key={key}
                  className={cn(
                    "flex items-center px-2 py-1.5 rounded-sm cursor-pointer",
                    selectedTagIndex === index
                      ? "bg-muted text-accent-foreground"
                      : "hover:bg-muted",
                  )}
                  onClick={() => applyTag(key)}
                >
                  <Icon icon={config.icon} className="size-4 mr-2" />
                  <span className="text-sm">{config.label}</span>
                </div>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Tab selectors - show when no active filter tab */}
      <AnimatePresence mode="wait">
        {!showFilters ? (
          <motion.div
            key="tabs"
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.2 }}
            className="px-2 pt-3"
          >
            <div className="flex flex-col">
              <p className="text-sm text-muted-foreground mb-2">
                I&apos;m looking for...
              </p>
              <Tabs
                defaultValue="all"
                value={activeTab}
                onValueChange={(value) => {
                  // Clear filters when changing tabs from the tabs UI
                  if (value !== activeTab) {
                    setActiveFilters({});
                  }
                  setActiveTab(value as SiteSearchTab);
                }}
              >
                <TabsList className="bg-transparent gap-2">
                  {Object.entries(tabConfig).map(([key, config]) => (
                    <TabsTrigger
                      key={key}
                      value={key}
                      className="data-[state=active]:bg-muted data-[state=active]:ring-2 data-[state=active]:ring-blue-600/20 data-[state=active]:border-blue-600 data-[state=active]:text- data-[state=active]:shadow-none bg-background border border-border hover:bg-muted cursor-pointer"
                    >
                      <div className="flex items-center gap-1.5">
                        <Icon icon={config.icon} className="size-3.5" />
                        {config.label}
                      </div>
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>
          </motion.div>
        ) : (
          /* Filters for the selected category */
          <motion.div
            key="filters"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 10 }}
            transition={{ duration: 0.2 }}
            className="px-1 pt-3 pb-2"
          >
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-1 text-sm text-muted-foreground">
                <span>Narrow it down</span>
              </div>

              <div className="flex flex-wrap gap-y-4">
                {currentTabConfig.filters.map((filter, filterIndex) => (
                  <motion.div
                    key={filter}
                    className="flex flex-col mr-4"
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{
                      duration: 0.2,
                      delay: filterIndex * 0.05,
                    }}
                  >
                    <span className="text-xs text-muted-foreground capitalize mb-1">
                      {filter.replace(/_/g, " ")}
                    </span>
                    <div className="flex flex-wrap gap-1">
                      {getFilterOptions(filter, activeTab).map(
                        (option, optionIndex) => (
                          <motion.div
                            key={`${filter}-${option.value}`}
                            initial={{ opacity: 0, scale: 0.9 }}
                            animate={{ opacity: 1, scale: 1 }}
                            transition={{
                              duration: 0.1,
                              delay: filterIndex * 0.05 + optionIndex * 0.02,
                            }}
                          >
                            <Button
                              variant={
                                activeFilters[filter] === option.value
                                  ? "default"
                                  : "outline"
                              }
                              size="sm"
                              className="h-7 text-xs"
                              onClick={() => {
                                if (activeFilters[filter] === option.value) {
                                  handleRemoveFilter(filter);
                                } else {
                                  handleAddFilter(filter, option.value);
                                }
                              }}
                            >
                              {option.label}
                            </Button>
                          </motion.div>
                        ),
                      )}
                    </div>
                  </motion.div>
                ))}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}
