import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { usePermission } from "@/hooks/use-permission";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import {
  DEFAULT_SOURCING_SORT,
  EMPTY_SOURCING_FILTERS,
  SOURCING_PAGE_SIZE,
  buildSourcingSearchInput,
  candidateFromLookup,
  candidateFromSearchResult,
  carrierIntelSupportsAutocomplete,
  carrierSourcingAvailability,
  countActiveFilters,
  dedupeByDotNumber,
  detectSourcingIntent,
  isLookupIntent,
  isSearchIntent,
  lookupInputForIntent,
  withImportedCarriers,
  type CarrierIntelProviderCapabilities,
  type SourcingCandidate,
  type SourcingFilters,
  type SourcingIntent,
} from "@/lib/carrier-sourcing";
import {
  CARRIER_SOURCING_SEARCH_KEY,
  searchCarrierSourcing,
  type CarrierSourcingPage,
  type CarrierSourcingSuggestion,
} from "@/lib/graphql/carrier-sourcing";
import { queries } from "@/lib/queries";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import type { CarrierSourcingSort } from "@trenova/graphql/generated/graphql";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo, useState } from "react";
import {
  CarrierDetailSheet,
  lookupQueryOptions,
  previewDepth,
  type CarrierSheetTarget,
} from "./carrier-detail-sheet";
import { ImportCarrierDialog, type ImportCandidate } from "./import-carrier-dialog";
import {
  SourcingResultList,
  SourcingResultListSkeleton,
  formatCarrierCount,
} from "./sourcing-result-list";
import { SourcingSearchBar } from "./sourcing-search-bar";
import {
  SourcingIntro,
  SourcingListSketch,
  SourcingNoMatches,
  SourcingNotConfigured,
  SourcingSearchUnsupported,
  type SourcingExample,
} from "./sourcing-states";
import { SourcingToolbar } from "./sourcing-toolbar";

function nextOffset(lastPage: CarrierSourcingPage, pages: CarrierSourcingPage[]) {
  const loaded = pages.length * SOURCING_PAGE_SIZE;
  return loaded < lastPage.total ? loaded : undefined;
}

export function CarrierSourcingWorkspace() {
  const t = useT();
  const [text, setText] = useState("");
  const [submitted, setSubmitted] = useState<SourcingIntent>({ kind: "empty" });
  const [filters, setFilters] = useState<SourcingFilters>(EMPTY_SOURCING_FILTERS);
  const [sort, setSort] = useState<CarrierSourcingSort>(DEFAULT_SOURCING_SORT);
  const [target, setTarget] = useState<CarrierSheetTarget | null>(null);
  const [importing, setImporting] = useState<ImportCandidate | null>(null);
  const [imported, setImported] = useState<Record<string, string>>({});

  const { allowed: canImportSourcing } = usePermission(Resource.CarrierSourcing, Operation.Import);
  const { allowed: canCreateCarrier } = usePermission(Resource.Carrier, Operation.Create);
  const { allowed: canReadIntel, isLoading: permissionsLoading } = usePermission(
    Resource.CarrierIntelligence,
    Operation.Read,
  );
  const canImport = canImportSourcing && canCreateCarrier;

  const settingsQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    enabled: canReadIntel,
  });
  const ruleLabels = useCarrierIntelRuleLabels(canReadIntel);

  const provider = useMemo<CarrierIntelProviderCapabilities | null>(() => {
    const info = settingsQuery.data?.carrierIntelProvider;
    return info
      ? { configured: info.configured, provider: info.provider, capabilities: info.capabilities }
      : null;
  }, [settingsQuery.data]);
  const availability = carrierSourcingAvailability(provider);
  const searchSupported = availability === "supported" || availability === "unknown";
  const depth = previewDepth(provider);

  const lookupMode = isLookupIntent(submitted);
  const searchInput = useMemo(
    () =>
      lookupMode || !searchSupported
        ? null
        : buildSourcingSearchInput(isSearchIntent(submitted) ? submitted.text : "", filters, sort),
    [filters, lookupMode, searchSupported, sort, submitted],
  );

  const searchQuery = useInfiniteQuery({
    queryKey: [CARRIER_SOURCING_SEARCH_KEY, searchInput],
    queryFn: ({ pageParam, signal }) => {
      if (!searchInput) {
        throw new Error("No search submitted");
      }
      return searchCarrierSourcing({ ...searchInput, offset: pageParam }, { signal });
    },
    initialPageParam: 0,
    getNextPageParam: nextOffset,
    enabled: searchInput !== null,
    staleTime: 10 * 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const lookupQuery = useQuery(
    lookupQueryOptions(
      isLookupIntent(submitted) ? lookupInputForIntent(submitted, depth) : {},
      isLookupIntent(submitted) && availability !== "not-configured",
    ),
  );

  const searchCandidates = useMemo(() => {
    const data = searchQuery.data;
    if (!data) {
      return [];
    }
    const fetchedAt = Math.floor(searchQuery.dataUpdatedAt / 1000) || null;
    const items = data.pages.flatMap((page) =>
      page.items.map((item) => candidateFromSearchResult(item, page.provider, fetchedAt)),
    );
    return dedupeByDotNumber(items).map((candidate) => withImportedCarriers(candidate, imported));
  }, [imported, searchQuery.data, searchQuery.dataUpdatedAt]);

  const lookupCandidate = useMemo(
    () =>
      lookupMode && lookupQuery.data
        ? withImportedCarriers(candidateFromLookup(lookupQuery.data), imported)
        : null,
    [imported, lookupMode, lookupQuery.data],
  );

  const submit = useCallback((value: string) => {
    setSubmitted(detectSourcingIntent(value));
  }, []);

  const clear = useCallback(() => {
    setText("");
    setSubmitted({ kind: "empty" });
  }, []);

  const openCandidate = useCallback((candidate: SourcingCandidate) => {
    setTarget({ dotNumber: candidate.dotNumber, seed: candidate });
  }, []);

  const openSuggestion = useCallback((suggestion: CarrierSourcingSuggestion) => {
    setTarget({ dotNumber: suggestion.dotNumber, seed: null });
  }, []);

  const startImport = useCallback((candidate: SourcingCandidate) => {
    setImporting({ dotNumber: candidate.dotNumber, legalName: candidate.legalName });
  }, []);

  const examples = useMemo<SourcingExample[]>(
    () =>
      searchSupported
        ? [
            {
              id: "state",
              label: t("Carriers based in Texas"),
              onSelect: () => setFilters({ ...EMPTY_SOURCING_FILTERS, state: "TX" }),
            },
            {
              id: "lane",
              label: t("Running Illinois → Georgia"),
              onSelect: () =>
                setFilters({
                  ...EMPTY_SOURCING_FILTERS,
                  originState: "IL",
                  destinationState: "GA",
                }),
            },
            {
              id: "hazmat",
              label: t("Hazmat carriers in Ohio"),
              onSelect: () =>
                setFilters({ ...EMPTY_SOURCING_FILTERS, state: "OH", screens: ["hazmat"] }),
            },
          ]
        : [],
    [searchSupported, t],
  );

  if (permissionsLoading || (canReadIntel && settingsQuery.isPending)) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-8 w-64" />
        <SourcingResultListSkeleton rows={4} />
      </div>
    );
  }

  if (availability === "not-configured") {
    return <SourcingNotConfigured />;
  }

  const providerName = carrierIntelProviderLabel(provider?.provider);
  const loadedPages = searchQuery.data?.pages ?? [];
  const total = loadedPages[0]?.total ?? 0;
  const hiddenByFilters = loadedPages.reduce((sum, page) => sum + page.filteredOut, 0);

  const summary = lookupMode
    ? lookupCandidate && !lookupCandidate.notFound
      ? formatCarrierCount(1, t)
      : null
    : searchInput && searchQuery.data
      ? hiddenByFilters > 0
        ? t(
            "{0} · {1} hidden by filters",
            formatCarrierCount(total, t),
            formatNumber(hiddenByFilters),
          )
        : formatCarrierCount(total, t)
      : null;

  const renderBody = () => {
    if (lookupMode) {
      if (lookupQuery.isPending) {
        return <SourcingResultListSkeleton rows={1} />;
      }
      if (lookupQuery.isError) {
        return (
          <IntelInlineError
            error={lookupQuery.error}
            title={t("The lookup didn't run")}
            onRetry={() => void lookupQuery.refetch()}
          />
        );
      }
      if (!lookupCandidate || lookupCandidate.notFound) {
        return (
          <EmptySheet
            title={t("No carrier found")}
            description={t(
              "{0} has no record for {1}. Check the number, or search by name instead.",
              lookupCandidate?.provider
                ? carrierIntelProviderLabel(lookupCandidate.provider)
                : providerName,
              submitted.kind === "mc"
                ? t("MC {0}", submitted.docketNumber)
                : submitted.kind === "dot"
                  ? t("USDOT {0}", submitted.dotNumber)
                  : "",
            )}
            sketch={<SourcingListSketch />}
          />
        );
      }
      return (
        <SourcingResultList
          candidates={[lookupCandidate]}
          canImport={canImport}
          onOpen={openCandidate}
          onImport={startImport}
        />
      );
    }

    if (!searchSupported) {
      return isSearchIntent(submitted) ? (
        <SourcingSearchUnsupported provider={provider?.provider ?? null} />
      ) : (
        <SourcingIntro examples={examples} searchSupported={false} />
      );
    }

    if (!searchInput) {
      return <SourcingIntro examples={examples} searchSupported />;
    }
    if (searchQuery.isPending) {
      return <SourcingResultListSkeleton />;
    }
    if (searchQuery.isError && !searchQuery.data) {
      return (
        <IntelInlineError
          error={searchQuery.error}
          title={t("The search didn't run")}
          onRetry={() => void searchQuery.refetch()}
        />
      );
    }

    return (
      <div className="flex flex-col gap-3">
        {searchCandidates.length === 0 ? (
          <SourcingNoMatches
            filtered={countActiveFilters(filters) > 0}
            onClearFilters={() => setFilters(EMPTY_SOURCING_FILTERS)}
          />
        ) : (
          <SourcingResultList
            candidates={searchCandidates}
            canImport={canImport}
            onOpen={openCandidate}
            onImport={startImport}
          />
        )}
        {searchQuery.isFetchNextPageError ? (
          <IntelInlineError
            error={searchQuery.error}
            title={t("The next page didn't load")}
            onRetry={() => void searchQuery.fetchNextPage()}
          />
        ) : searchQuery.hasNextPage ? (
          <div className="flex justify-center">
            <Button
              type="button"
              variant="ghost"
              isLoading={searchQuery.isFetchingNextPage}
              onClick={() => void searchQuery.fetchNextPage()}
            >
              {t("Load more")}
            </Button>
          </div>
        ) : null}
      </div>
    );
  };

  return (
    <div className="flex flex-col gap-3">
      <SourcingSearchBar
        value={text}
        onValueChange={setText}
        onSubmit={submit}
        onClear={clear}
        onPickSuggestion={openSuggestion}
        autocompleteEnabled={
          availability === "unknown" || carrierIntelSupportsAutocomplete(provider)
        }
        isBusy={
          (lookupMode && lookupQuery.isFetching) ||
          (searchInput !== null && searchQuery.isFetching && !searchQuery.isFetchingNextPage)
        }
      />
      <SourcingToolbar
        filters={filters}
        onFiltersChange={setFilters}
        sort={sort}
        onSortChange={setSort}
        showFilters={searchSupported && !lookupMode}
        summary={
          summary ? (
            <>
              {summary}
              {provider?.configured ? (
                <span className="text-muted-foreground/70"> · {providerName}</span>
              ) : null}
            </>
          ) : null
        }
      />
      {renderBody()}
      <CarrierDetailSheet
        target={target}
        provider={provider}
        imported={imported}
        canImport={canImport}
        ruleLabels={ruleLabels}
        onOpenChange={(open) => {
          if (!open) {
            setTarget(null);
          }
        }}
        onImport={startImport}
      />
      <ImportCarrierDialog
        candidate={importing}
        canEnrollMonitoring={provider?.configured ?? true}
        onOpenChange={(open) => {
          if (!open) {
            setImporting(null);
          }
        }}
        onImported={(dotNumber, carrier) =>
          setImported((current) => ({ ...current, [dotNumber]: carrier.id }))
        }
      />
    </div>
  );
}
