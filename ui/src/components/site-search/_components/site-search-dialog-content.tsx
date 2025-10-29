import { CommandGroup, CommandList } from "@/components/ui/command";
import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Spinner } from "@/components/ui/shadcn-io/spinner";
import { SearchEntityType, SearchHit } from "@/types/search";
import { faBox, faSearch, faTruck } from "@fortawesome/pro-regular-svg-icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { ResultItemOuterContainer, ShipmentResultItem } from "./result-items";
import { ShipmentSearchPreview } from "./shipment/shipment-preview";

export function SiteSearchDialogContent({
  searchQuery,
  isLoading,
  setOpen,
  setSearchQuery,
  searchResults,
  activeTab,
}: {
  searchQuery: string;
  isLoading: boolean;
  setOpen: (open: boolean) => void;
  setSearchQuery: (query: string) => void;
  searchResults?: SearchHit[];
  activeTab: SearchEntityType;
}) {
  const navigate = useNavigate();
  const [previewId, setPreviewId] = useState<string | undefined>(undefined);
  const listRef = useRef<HTMLDivElement | null>(null);

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

  const shipments = useMemo(
    () =>
      (searchResults || []).filter(
        (r) => r.entityType === SearchEntityType.Shipment,
      ),
    [searchResults],
  );

  const effectivePreviewId = useMemo(() => {
    if (
      (activeTab === SearchEntityType.All ||
        activeTab === SearchEntityType.Shipment) &&
      shipments.length > 0
    ) {
      return previewId ?? shipments[0].id;
    }
    return undefined;
  }, [activeTab, shipments, previewId]);

  useEffect(() => {
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
  }, [listRef]);

  if (!searchQuery) {
    return null;
  }

  if (isLoading) {
    return (
      <div className="flex flex-col justify-center items-center h-60 gap-2">
        <Spinner variant="bars" className="size-8 text-primary" />

        <div className="flex flex-col items-center text-center">
          <h3 className="text-lg font-semibold text-foreground">
            Searching for{" "}
            <span className="font-mono">&quot;{searchQuery}&quot;</span>...
          </h3>
          <p className="text-sm text-muted-foreground">
            This may take a few seconds...
          </p>
        </div>
      </div>
    );
  }

  if (!isLoading && typeof searchResults === "undefined") {
    const entityLabel =
      activeTab === SearchEntityType.All
        ? "shipments and customers"
        : activeTab === SearchEntityType.Shipment
          ? "shipments"
          : "customers";
    return (
      <div className="flex flex-col justify-center items-center h-60 gap-2">
        <Spinner variant="infinite" className="size-8 text-primary" />
        <div className="flex flex-col items-center text-center">
          <p className="text-sm font-medium text-foreground">
            Keep typing to search {entityLabel}...
          </p>
        </div>
      </div>
    );
  }

  if (searchResults?.length === 0) {
    return (
      <div className="flex justify-center items-center h-full">
        <EmptyState
          title="No results found"
          description="Try adjusting your search query"
          icons={[faSearch, faBox, faTruck]}
          className="size-full border-none bg-transparent"
        />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-[1.2fr_1fr] gap-2">
      <CommandList ref={listRef}>
        {shipments.length > 0 && (
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
                <ShipmentResultItem result={result} searchQuery={searchQuery} />
              </ResultItemOuterContainer>
            ))}
          </CommandGroup>
        )}
      </CommandList>
      <ScrollArea className="hidden md:block border-l border-border p-2 pr-3 max-h-[300px]">
        {(activeTab === SearchEntityType.Shipment ||
          activeTab === SearchEntityType.All) &&
        shipments.length > 0 ? (
          <ShipmentSearchPreview shipmentId={effectivePreviewId} />
        ) : null}
      </ScrollArea>
    </div>
  );
}
