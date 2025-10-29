"use no memo";
import { useSearch } from "@/hooks/use-search";
import { cn } from "@/lib/utils";
import { SearchEntityType } from "@/types/search";
import React, { useState } from "react";
import { CommandDialog, CommandInput } from "../ui/command";
import { SiteSearchDialogContent } from "./_components/site-search-dialog-content";

export function SearchDialog() {
  const [open, setOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<SearchEntityType>(
    SearchEntityType.All,
  );
  const { searchQuery, setSearchQuery, searchResults, isLoading } =
    useSearch(activeTab);

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "j" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  return (
    <>
      <p className="text-muted-foreground text-sm">
        Press{" "}
        <kbd className="bg-muted text-muted-foreground pointer-events-none inline-flex h-5 items-center gap-1 rounded border px-1.5 font-mono text-[10px] font-medium opacity-100 select-none">
          <span className="text-xs">⌘</span>J
        </kbd>
      </p>
      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        shouldFilter={false}
        contentClassName={cn(
          "z-50 grid w-full max-w-[calc(100%-2rem)] gap-4 border duration-200 sm:max-w-3xl",
          "rounded-xl border-none bg-clip-padding shadow-2xl ring-4",
          "ring-neutral-200/80 dark:bg-neutral-900 dark:ring-neutral-800",
        )}
      >
        <CommandInput
          placeholder={
            activeTab === "all"
              ? "Search shipments and customers..."
              : activeTab === "shipment"
                ? "Search shipments..."
                : "Search customers..."
          }
          value={searchQuery}
          onValueChange={setSearchQuery}
        />
        {!searchQuery && (
          <div className="flex flex-col gap-2 p-2">
            <h4 className="text-sm font-medium text-muted-foreground">
              I&apos;m looking for...
            </h4>
            <div className="flex items-center gap-1.5 text-xs">
              {(Object.values(SearchEntityType) as SearchEntityType[])
                .sort((a, b) => a.localeCompare(b))
                .map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setActiveTab(tab)}
                    className={cn(
                      "px-2 py-0.5 rounded-md border",
                      activeTab === tab
                        ? "bg-muted text-foreground border-border"
                        : "bg-background text-muted-foreground border-border hover:bg-muted",
                    )}
                  >
                    {tab === "all"
                      ? "All"
                      : tab === SearchEntityType.Shipment
                        ? "Shipments"
                        : "Customers"}
                  </button>
                ))}
            </div>
          </div>
        )}
        <SiteSearchDialogContent
          searchQuery={searchQuery}
          isLoading={isLoading}
          setOpen={setOpen}
          setSearchQuery={setSearchQuery}
          searchResults={searchResults}
          activeTab={activeTab}
        />
      </CommandDialog>
    </>
  );
}
