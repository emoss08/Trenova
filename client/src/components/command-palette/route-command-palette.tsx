import Highlight from "@/components/highlight";
import type { SidebarLink } from "@/components/sidebar-nav";
import {
  CommandDialog,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Kbd, KbdGroup } from "@/components/ui/kbd";
import { adminLinks, navigationConfig } from "@/config/navigation.config";
import { useDebounce } from "@/hooks/use-debounce";
import { useFilteredNavigation } from "@/hooks/use-filtered-navigation";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { usePermissionStore } from "@/stores/permission-store";
import { Operation } from "@/types/permission";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useQuery } from "@tanstack/react-query";
import { ArrowDown, ArrowRight, ArrowUp, CornerDownLeft, Plus, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { ShipmentSearchPreview } from "./_components/shipment/shipment-preview";
import { buildRouteCommandGroups, buildSuggestedCreateCommands } from "./route-command-data";
import {
  filterMentionOptions,
  getMentionState,
  getSearchEntityOption,
  resolveEntityAlias,
  stripMentionToken,
  type SearchableEntityType,
} from "./search-entity-filter";
import { SearchResultItem } from "./search-result-items";
import { SearchEmpty, SearchError, SearchKeepTyping, SearchLoading } from "./search-states";

function normalizeLinkPath(path: string): string {
  if (!path || path === "#") {
    return "";
  }

  const trimmed = path.trim();
  if (!trimmed || trimmed === "#") {
    return "";
  }

  const withLeadingSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  const normalized = withLeadingSlash.replace(/\/+$/, "");
  return normalized || "/";
}

export function RouteCommandPalette() {
  const navigate = useNavigate();
  const location = useLocation();
  const filteredModules = useFilteredNavigation();
  const open = useCommandPaletteStore((state) => state.open);
  const setOpen = useCommandPaletteStore((state) => state.setOpen);
  const toggleOpen = useCommandPaletteStore((state) => state.toggleOpen);
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const canAccessRoute = usePermissionStore((state) => state.canAccessRoute);
  const [searchValue, setSearchValue] = useState("");
  const [recordEntityFilter, setRecordEntityFilter] = useState<SearchableEntityType | null>(null);
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionText, setMentionText] = useState("");
  const [mentionIndex, setMentionIndex] = useState(0);
  const [previewId, setPreviewId] = useState<string | undefined>(undefined);
  const listRef = useRef<HTMLDivElement | null>(null);
  const accessibleAdminLinks = useMemo(
    () =>
      adminLinks.filter((link: SidebarLink) => {
        if (link.disabled) {
          return false;
        }

        if (!manifest) {
          return false;
        }

        if (manifest.isPlatformAdmin) {
          return true;
        }

        if (link.adminOnly && !manifest.isOrgAdmin) {
          return false;
        }

        if (link.resource) {
          return hasPermission(link.resource, link.requiredOperation ?? Operation.Read);
        }

        const normalizedPath = normalizeLinkPath(link.href);
        if (!normalizedPath) {
          return false;
        }

        return canAccessRoute(normalizedPath) || canAccessRoute(link.href);
      }),
    [canAccessRoute, hasPermission, manifest],
  );
  const routeGroups = useMemo(
    () => buildRouteCommandGroups(filteredModules, accessibleAdminLinks),
    [accessibleAdminLinks, filteredModules],
  );
  const suggestedCommands = useMemo(
    () =>
      buildSuggestedCreateCommands(
        navigationConfig.quickActions ?? [],
        routeGroups,
        (resource, operation) => hasPermission(resource, operation),
      ),
    [hasPermission, routeGroups],
  );
  const hasQuery = searchValue.trim().length > 0;
  const activeEntityOption = useMemo(
    () => getSearchEntityOption(recordEntityFilter),
    [recordEntityFilter],
  );
  const filteredEntityOptions = useMemo(() => filterMentionOptions(mentionText), [mentionText]);
  const debouncedSearchValue = useDebounce(searchValue, 200);
  const showSuggestedCommands = hasQuery;
  const remoteSearchQuery = useQuery({
    queryKey: ["global-search", debouncedSearchValue, recordEntityFilter],
    queryFn: () =>
      apiService.globalSearchService.search(
        debouncedSearchValue,
        10,
        recordEntityFilter ? [recordEntityFilter] : undefined,
      ),
    enabled: open && debouncedSearchValue.trim().length >= 2,
    staleTime: 15_000,
  });
  const remoteGroups = useMemo(
    () => remoteSearchQuery.data?.groups ?? [],
    [remoteSearchQuery.data?.groups],
  );
  const remoteQueryReady = hasQuery && debouncedSearchValue.trim().length >= 2;
  const shipmentHits = useMemo(
    () => remoteGroups.find((g) => g.entityType === "shipment")?.hits ?? [],
    [remoteGroups],
  );
  const showPreview = shipmentHits.length > 0 && remoteQueryReady && !remoteSearchQuery.isFetching;
  const effectivePreviewId = showPreview ? (previewId ?? shipmentHits[0]?.id) : undefined;

  const handleNavigate = (href: string) => {
    setSearchValue("");
    setRecordEntityFilter(null);
    setMentionOpen(false);
    setMentionText("");
    setMentionIndex(0);
    setPreviewId(undefined);
    setOpen(false);
    void navigate(href);
  };

  const handleSelectEntityFilter = (entityType: SearchableEntityType) => {
    setRecordEntityFilter(entityType);
    setSearchValue((currentValue) => stripMentionToken(currentValue));
    setMentionOpen(false);
    setMentionText("");
    setMentionIndex(0);
  };

  const handleSearchValueChange = (value: string) => {
    const committedMention = value.match(/@([a-zA-Z]+)\s$/);
    if (committedMention) {
      const committedFilter = resolveEntityAlias(committedMention[1] ?? "");
      if (committedFilter) {
        setRecordEntityFilter(committedFilter);
        setSearchValue(stripMentionToken(value));
        setMentionOpen(false);
        setMentionText("");
        setMentionIndex(0);
        return;
      }
    }

    if (value.length === 0) {
      setRecordEntityFilter(null);
    }

    const mentionState = getMentionState(value);
    setMentionOpen(mentionState.mentionOpen);
    setMentionText(mentionState.mentionText);
    setMentionIndex(0);
    setSearchValue(value);
  };

  useEffect(() => {
    setSearchValue("");
    setRecordEntityFilter(null);
    setMentionOpen(false);
    setMentionText("");
    setMentionIndex(0);
    setPreviewId(undefined);
  }, [location.pathname, location.search, location.hash]);

  useHotkey(
    "Mod+K",
    () => {
      toggleOpen();
    },
    {
      ignoreInputs: true,
      preventDefault: true,
    },
  );

  const handleShipmentPreview = useCallback((id: string) => {
    setPreviewId(id);
  }, []);

  useEffect(() => {
    if (!open || !showPreview) return;
    const root = listRef.current;
    if (!root) return;

    const syncPreview = () => {
      const selected = root.querySelector(
        '[data-slot="command-item"][data-selected="true"]',
      ) as HTMLElement | null;
      const entityType = selected?.dataset.entityType;
      const id = selected?.dataset.id;
      if (entityType === "shipment" && id) {
        setPreviewId(id);
      }
    };

    const observer = new MutationObserver(syncPreview);
    observer.observe(root, {
      attributes: true,
      attributeFilter: ["data-selected"],
      subtree: true,
    });

    return () => observer.disconnect();
  }, [open, showPreview]);

  const remoteResultGroups =
    remoteQueryReady && !remoteSearchQuery.isFetching && !remoteSearchQuery.isError
      ? remoteGroups.map((group) => (
          <CommandGroup key={group.entityType} heading={group.label}>
            {group.hits.map((hit) => (
              <SearchResultItem
                key={`${group.entityType}:${hit.id}`}
                hit={hit}
                searchValue={searchValue}
                onSelect={() => handleNavigate(hit.href)}
                onPreview={hit.entityType === "shipment" ? handleShipmentPreview : undefined}
              />
            ))}
          </CommandGroup>
        ))
      : null;

  const remoteStatusIndicator = (() => {
    if (!hasQuery || !recordEntityFilter) return null;
    if (debouncedSearchValue.trim().length < 2) return <SearchKeepTyping />;
    if (remoteSearchQuery.isFetching) return <SearchLoading query={debouncedSearchValue} />;
    if (remoteSearchQuery.isError) return <SearchError />;
    if (remoteGroups.length === 0) return <SearchEmpty />;
    return null;
  })();

  return (
    <CommandDialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) {
          setSearchValue("");
          setRecordEntityFilter(null);
          setMentionOpen(false);
          setMentionText("");
          setMentionIndex(0);
          setPreviewId(undefined);
        }
      }}
      title="Command Palette"
      description="Search for routes, commands, and synced records."
      className={cn(
        "z-50 grid w-full max-w-4xl gap-4 overflow-visible border duration-200 sm:max-w-4xl",
        "rounded-md border-none bg-clip-padding shadow-2xl ring-4",
        "ring-neutral-200/80 dark:bg-neutral-900 dark:ring-neutral-800",
      )}
      commandProps={{
        className:
          "overflow-visible **:data-[slot=command-input]:h-12 **:data-[slot=command-input-wrapper]:h-12",
        filter: (value, search, keywords = []) => {
          const normalizedSearch = search.trim().toLowerCase();
          if (!normalizedSearch) return 1;

          const haystack = `${value} ${keywords.join(" ")}`.toLowerCase();
          const terms = normalizedSearch.split(/\s+/).filter(Boolean);
          const matchesAllTerms = terms.every((term) => haystack.includes(term));

          return matchesAllTerms ? 1 : 0;
        },
      }}
    >
      <div className="flex flex-col">
        <div className="relative">
          <CommandInput
            className={cn("h-12", activeEntityOption ? "pl-24" : undefined)}
            placeholder={
              activeEntityOption
                ? `Search ${activeEntityOption.label.toLowerCase()}...`
                : "Search routes, commands, or records..."
            }
            value={searchValue}
            onValueChange={handleSearchValueChange}
            onKeyDown={(event) => {
              if (!mentionOpen || filteredEntityOptions.length === 0) {
                return;
              }

              if (event.key === "ArrowDown") {
                event.preventDefault();
                setMentionIndex((currentIndex) => {
                  return (currentIndex + 1) % filteredEntityOptions.length;
                });
                return;
              }

              if (event.key === "ArrowUp") {
                event.preventDefault();
                setMentionIndex((currentIndex) => {
                  return (
                    (currentIndex - 1 + filteredEntityOptions.length) % filteredEntityOptions.length
                  );
                });
                return;
              }

              if (event.key === "Enter") {
                const target = filteredEntityOptions[mentionIndex];
                if (!target) {
                  return;
                }

                event.preventDefault();
                handleSelectEntityFilter(target.key);
                return;
              }

              if (event.key === "Escape") {
                event.preventDefault();
                setMentionOpen(false);
                setMentionText("");
                setMentionIndex(0);
              }
            }}
          />
          {activeEntityOption && (
            <div className="absolute top-1/2 left-8 -translate-y-1/2">
              <button
                type="button"
                onClick={() => setRecordEntityFilter(null)}
                className="inline-flex h-5 items-center gap-1 rounded-full border bg-muted px-2 py-0 text-2xs font-medium text-foreground transition-colors hover:bg-muted/80"
                aria-label={`Clear ${activeEntityOption.label} record filter`}
              >
                <span>{activeEntityOption.label}</span>
                <X className="size-3" />
              </button>
            </div>
          )}
          {mentionOpen && filteredEntityOptions.length > 0 && (
            <div className="absolute top-11 left-2 z-50 w-52 rounded-lg border bg-popover p-2 shadow-lg">
              <div className="px-2 pb-1 text-2xs font-medium tracking-[0.18em] text-muted-foreground uppercase">
                Filter records
              </div>
              <div className="flex flex-col gap-1">
                {filteredEntityOptions.map((option, index) => (
                  <button
                    key={option.key}
                    type="button"
                    onClick={() => handleSelectEntityFilter(option.key)}
                    className={cn(
                      "rounded-md px-3 py-2 text-left text-sm transition-colors",
                      index === mentionIndex
                        ? "bg-accent text-accent-foreground"
                        : "text-foreground hover:bg-muted",
                    )}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        <div
          className={cn(
            "grid",
            showPreview
              ? "h-[min(60vh,36rem)] grid-cols-[1.2fr_1fr]"
              : "max-h-[min(60vh,36rem)] grid-cols-1",
          )}
        >
          <div className="overflow-y-auto" ref={listRef}>
            <CommandList className="max-h-none">
              {!recordEntityFilter && (
                <>
                  {routeGroups.map((group) => (
                    <CommandGroup key={group.id} heading={group.label}>
                      {group.items.map((item) => (
                        <CommandItem
                          key={item.id}
                          className="group"
                          value={`${item.title} ${item.subtitle} ${item.keywords.join(" ")}`}
                          onSelect={() => handleNavigate(item.href)}
                        >
                          <item.icon className="size-4" />
                          <div className="flex flex-1 flex-col">
                            <Highlight text={item.title} highlight={searchValue} />
                            <Highlight
                              text={item.subtitle}
                              highlight={searchValue}
                              className="text-2xs text-muted-foreground"
                            />
                          </div>
                          <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  ))}
                  {showSuggestedCommands && suggestedCommands.length > 0 && (
                    <CommandGroup heading="Suggested commands">
                      {suggestedCommands.map((item) => (
                        <CommandItem
                          key={item.id}
                          className="group"
                          value={`${item.label} ${item.description} ${item.keywords.join(" ")}`}
                          onSelect={() => handleNavigate(item.href)}
                        >
                          <Plus className="size-4" />
                          <div className="flex flex-1 flex-col">
                            <Highlight text={item.label} highlight={searchValue} />
                            <Highlight
                              text={item.description}
                              highlight={searchValue}
                              className="text-2xs text-muted-foreground"
                            />
                          </div>
                          <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  )}
                </>
              )}
              {remoteResultGroups}
            </CommandList>
            {remoteStatusIndicator}
          </div>
          {showPreview && (
            <div className="hidden max-h-[min(60vh,36rem)] overflow-hidden border-l border-border md:block">
              <ShipmentSearchPreview shipmentId={effectivePreviewId} />
            </div>
          )}
        </div>

        <div className={cn("flex items-center gap-6 border-t bg-muted/30 px-4 py-2 text-xs text-muted-foreground", (mentionOpen || (recordEntityFilter && remoteGroups.length === 0)) && "hidden")}>
          <div className="flex items-center gap-2">
            <KbdGroup>
              <Kbd>
                <ArrowUp className="size-3" />
              </Kbd>
              <Kbd>
                <ArrowDown className="size-3" />
              </Kbd>
            </KbdGroup>
            <span>to navigate</span>
          </div>
          <div className="flex items-center gap-2">
            <Kbd>
              <CornerDownLeft className="size-3" />
            </Kbd>
            <span>to select</span>
          </div>
          <div className="flex items-center gap-2">
            <Kbd>Esc</Kbd>
            <span>to close</span>
          </div>
        </div>
      </div>
    </CommandDialog>
  );
}
