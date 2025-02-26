import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { SearchInputProps, SiteSearchTab } from "@/types/search";
import {
  faCommand,
  faSearch,
  faXmark,
} from "@fortawesome/pro-regular-svg-icons";
import { AnimatePresence, motion } from "framer-motion";
import React, { useEffect, useState } from "react";
import { getFilterOptions, tabConfig } from "./site-search-filter-options";

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

  // Update filter visibility when tab changes
  useEffect(() => {
    setShowFilters(activeTab !== "all");
  }, [activeTab]);

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

  // Get config for current tab, use the enhanced config
  const currentTabConfig = tabConfig[activeTab] || tabConfig.all;

  return (
    <>
      <div className="relative border-b border-border">
        <div className="pointer-events-none absolute inset-y-0 gap-1.5 left-0 flex items-center pl-2 text-xs text-muted-foreground">
          <Icon icon={faSearch} className="size-4" />
        </div>

        <div className="flex items-center min-h-[48px] w-full pl-8 pr-12 py-2">
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
                  "mr-1.5 text-xs flex items-center px-1.5 py-0.5 rounded-md",
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

          {/* Filter badges */}
          <AnimatePresence>
            {Object.entries(activeFilters).map(([filter, filterValue]) => (
              <motion.div
                key={filter}
                initial={{ opacity: 0, scale: 0.8, x: -10 }}
                animate={{ opacity: 1, scale: 1, x: 0 }}
                exit={{ opacity: 0, scale: 0.8, x: -10 }}
                transition={{
                  type: "spring",
                  stiffness: 500,
                  damping: 30,
                  delay: 0.05,
                }}
                className="mr-1.5 text-xs bg-accent/50 text-accent-foreground flex items-center gap-1 px-1.5 py-0.5 rounded-md"
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
                  <Icon icon={faXmark} className="size-3" />
                </button>
              </motion.div>
            ))}
          </AnimatePresence>

          <input
            ref={inputRef}
            placeholder={
              activeTab === "all"
                ? "Search for anything"
                : `Search in ${currentTabConfig.label.toLowerCase()}`
            }
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="flex h-full w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground border-none"
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

      {/* Tab selectors - show when no active filter tab */}
      <AnimatePresence mode="wait">
        {!showFilters ? (
          <motion.div
            key="tabs"
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.2 }}
            className="px-1 pt-3"
          >
            <div className="flex flex-col">
              <p className="text-sm text-muted-foreground mb-2">
                I&apos;m looking for...
              </p>
              <Tabs
                defaultValue="all"
                value={activeTab}
                onValueChange={(value) => setActiveTab(value as SiteSearchTab)}
              >
                <TabsList className="bg-transparent gap-2">
                  {Object.entries(tabConfig).map(([key, config]) => (
                    <TabsTrigger
                      key={key}
                      value={key}
                      className="data-[state=active]:bg-foreground data-[state=active]:text-background data-[state=active]:shadow-none bg-muted cursor-pointer"
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
