"use no memo";
import { useSearch } from "@/hooks/use-search";
import { cn } from "@/lib/utils";
import { SearchEntityType, type SearchHit } from "@/types/search";
import React, { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../ui/command";
import { ScrollArea } from "../ui/scroll-area";
import {
  ResultItemOuterContainer,
  ShipmentResultItem,
} from "./_components/result-items";
import { ShipmentSearchPreview } from "./_components/shipment/shipment-preview";

export function SearchDialog() {
  const [open, setOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<SearchEntityType>(
    SearchEntityType.All,
  );
  const [previewId, setPreviewId] = useState<string | undefined>(undefined);
  const listRef = useRef<HTMLDivElement | null>(null);
  const navigate = useNavigate();

  const {
    searchQuery,
    setSearchQuery,
    searchResults,
    isLoading,
    isError,
    error,
  } = useSearch(activeTab);

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

  const shipments = useMemo(
    () =>
      (searchResults || []).filter(
        (r) => r.entityType === SearchEntityType.Shipment,
      ),
    [searchResults],
  );
  const customers = useMemo(
    () =>
      (searchResults || []).filter(
        (r) => r.entityType === SearchEntityType.Customer,
      ),
    [searchResults],
  );

  React.useEffect(() => {
    console.debug("[SearchDialog] state", {
      activeTab,
      searchQuery,
      isLoading,
      results: searchResults?.length ?? 0,
      shipments: shipments.length,
      customers: customers.length,
    });
  }, [activeTab, searchQuery, isLoading, searchResults, shipments, customers]);

  React.useEffect(() => {
    if (isError) {
      console.error("[SearchDialog] search error", error);
    }
  }, [isError, error]);

  React.useEffect(() => {
    if (
      (activeTab === SearchEntityType.All ||
        activeTab === SearchEntityType.Shipment) &&
      shipments.length > 0
    ) {
      setPreviewId((prev) => prev ?? shipments[0].id);
    } else if (activeTab === SearchEntityType.Customer) {
      setPreviewId(undefined);
    }
  }, [activeTab, shipments]);

  React.useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
      setTimeout(() => {
        const root = listRef.current;
        const selected = root?.querySelector(
          '[data-slot="command-item"][data-selected="true"]',
        ) as HTMLElement | null;
        const entityType = selected?.dataset.entityType;
        const id = selected?.dataset.id;
        if (entityType === "shipment" && id) {
          setPreviewId(id);
        }
      }, 0);
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open]);

  const handleSelect = (result: SearchHit) => {
    let link = "/search";
    if (result.entityType === SearchEntityType.Shipment) {
      link = `/shipments/management?entityId=${result.id}&modalType=edit`;
    } else if (result.entityType === SearchEntityType.Customer) {
      link = `/billing/configurations/customers?entityId=${result.id}&modalType=edit`;
    }
    setOpen(false);
    navigate(link);
    setSearchQuery("");
  };

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
        <div className="px-2 pt-2">
          <div className="flex items-center gap-1.5 text-xs">
            {(Object.values(SearchEntityType) as SearchEntityType[]).map(
              (tab) => (
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
              ),
            )}
          </div>
        </div>
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
        <div className="grid grid-cols-1 md:grid-cols-[1.2fr_1fr] gap-2">
          <CommandList ref={listRef}>
            {!searchQuery && (
              <CommandEmpty>Start typing to search...</CommandEmpty>
            )}
            {searchQuery && isLoading && (
              <CommandEmpty>Searching...</CommandEmpty>
            )}
            {searchQuery &&
              !isLoading &&
              (!searchResults || searchResults.length === 0) && (
                <CommandEmpty>No results found.</CommandEmpty>
              )}

            {searchQuery && !isLoading && shipments.length > 0 && (
              <CommandGroup heading="Shipments">
                {shipments.map((result) => (
                  <ResultItemOuterContainer
                    key={result.id}
                    value={result.id}
                    data-id={result.id}
                    data-entity-type="shipment"
                    onMouseEnter={() => setPreviewId(result.id)}
                    onFocus={() => setPreviewId(result.id)}
                    onSelect={() => handleSelect(result)}
                  >
                    <ShipmentResultItem
                      result={result}
                      searchQuery={searchQuery}
                    />
                  </ResultItemOuterContainer>
                ))}
              </CommandGroup>
            )}

            {searchQuery && !isLoading && customers.length > 0 && (
              <CommandGroup heading="Customers">
                {customers.map((result) => (
                  <CommandItem
                    key={result.id}
                    value={result.id}
                    onSelect={() => handleSelect(result)}
                  >
                    <div className="flex flex-col min-w-0">
                      <span className="text-sm font-medium truncate">
                        {result.metadata?.customerName || result.title}
                      </span>
                      <span className="text-2xs text-muted-foreground truncate">
                        {result.metadata?.customerCode || result.subtitle}
                      </span>
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>

          <ScrollArea className="hidden md:block border-l border-border p-2 pr-3 max-h-[300px]">
            {(activeTab === SearchEntityType.Shipment ||
              activeTab === SearchEntityType.All) &&
            shipments.length > 0 ? (
              <ShipmentSearchPreview shipmentId={previewId} />
            ) : (
              <div className="h-full w-full flex items-center justify-center text-2xs text-muted-foreground">
                Select a result to preview
              </div>
            )}
          </ScrollArea>
        </div>
      </CommandDialog>
    </>
  );
}
