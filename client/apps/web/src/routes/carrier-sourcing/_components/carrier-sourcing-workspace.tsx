import { useT } from "@trenova/shared/i18n/use-t";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { usePermission } from "@/hooks/use-permission";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import {
  carrierIntelSupportsAutocomplete,
  carrierSourcingAvailability,
  type CarrierIntelProviderCapabilities,
} from "@/lib/carrier-sourcing";
import {
  CARRIER_SOURCING_SEARCH_KEY,
  searchCarrierSourcing,
  type CarrierSourcingSuggestion,
} from "@/lib/graphql/carrier-sourcing";
import { queries } from "@/lib/queries";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, ScanSearchIcon, SearchXIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";
import { ImportCarrierDialog, type ImportCandidate } from "./import-carrier-dialog";
import { ProspectLookup, type ProspectLookupRequest } from "./prospect-lookup";
import type { SourcingCandidate } from "./sourcing-result-card";
import { SourcingResults, SourcingResultsSketch } from "./sourcing-results";
import {
  SOURCING_PAGE_SIZE,
  toSourcingSearchInput,
  type LookupKind,
  type SourcingSearchFormValues,
} from "./sourcing-schema";
import { SourcingSearchForm } from "./sourcing-search-form";

const SOURCING_MODES = ["search", "lookup"] as const;
type SourcingMode = (typeof SOURCING_MODES)[number];

const modeParser = parseAsStringLiteral(SOURCING_MODES).withDefault("search");

type SubmittedSearch = {
  values: SourcingSearchFormValues;
  offset: number;
};

export type SearchUnavailableProps = {
  availability: "not-configured" | "unsupported";
  providerName: string;
  onLookup: () => void;
};

export function SearchUnavailable({
  availability,
  providerName,
  onLookup,
}: SearchUnavailableProps) {
  const t = useT();

  if (availability === "not-configured") {
    return (
      <EmptySheet
        title={t("No carrier intelligence provider is connected")}
        description={t(
          "Connect CarrierOK to search the carrier market, or the free FMCSA QCMobile service to look carriers up by USDOT number before importing them.",
        )}
        sketch={<SourcingResultsSketch />}
        action={
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open integrations")}
          </Button>
        }
      />
    );
  }

  return (
    <div data-testid="sourcing-search-unsupported">
      <EmptySheet
        title={t("Search not supported by {0}", providerName)}
        description={t(
          "{0} can look carriers up by USDOT or MC number but cannot search the market by name, state or lane. Look a carrier up by number instead, or connect a provider that supports search.",
          providerName,
        )}
        sketch={<SourcingResultsSketch />}
        action={
          <div className="flex flex-wrap items-center justify-center gap-2">
            <Button type="button" size="sm" onClick={onLookup}>
              <ScanSearchIcon className="size-3.5" />
              {t("Look up by number")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              nativeButton={false}
              render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
            >
              <SearchXIcon className="size-3.5" />
              {t("Change provider")}
            </Button>
          </div>
        }
      />
    </div>
  );
}

export function CarrierSourcingWorkspace() {
  const t = useT();
  const [mode, setMode] = useQueryState("mode", modeParser);
  const [submitted, setSubmitted] = useState<SubmittedSearch | null>(null);
  const [lookupRequest, setLookupRequest] = useState<ProspectLookupRequest | null>(null);
  const [lookupPrefill, setLookupPrefill] = useState<{ kind: LookupKind; number: string } | null>(
    null,
  );
  const [importing, setImporting] = useState<ImportCandidate | null>(null);

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
  const providerName = carrierIntelProviderLabel(provider?.provider);
  const searchInput = submitted ? toSourcingSearchInput(submitted.values, submitted.offset) : null;

  const searchQuery = useQuery({
    queryKey: [CARRIER_SOURCING_SEARCH_KEY, searchInput],
    queryFn: ({ signal }) => {
      if (!searchInput) {
        throw new Error("No search submitted");
      }
      return searchCarrierSourcing(searchInput, { signal });
    },
    enabled: searchInput !== null && availability !== "unsupported",
    staleTime: 10 * 60_000,
    retry: false,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });

  const openLookupFor = useCallback(
    (suggestion: CarrierSourcingSuggestion) => {
      setLookupPrefill({ kind: "dot", number: suggestion.dotNumber });
      setLookupRequest({ input: { dotNumber: suggestion.dotNumber }, requestedAt: Date.now() });
      void setMode("lookup");
    },
    [setMode],
  );

  const handleImport = useCallback((candidate: SourcingCandidate) => {
    setImporting({ dotNumber: candidate.dotNumber, legalName: candidate.legalName });
  }, []);

  if (permissionsLoading || (canReadIntel && settingsQuery.isPending)) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-8 w-72" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-28 w-full" />
      </div>
    );
  }

  const blocked = availability === "not-configured";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <SegmentedControl<SourcingMode>
          items={[
            { value: "search", label: t("Search the market") },
            { value: "lookup", label: t("Look up by number") },
          ]}
          value={mode}
          onValueChange={(next) => void setMode(next === "search" ? null : next)}
          aria-label={t("Sourcing mode")}
        />
        {provider?.configured ? (
          <span className="text-muted-foreground text-xs">
            {t("Results from {0}", providerName)}
          </span>
        ) : null}
      </div>
      {blocked ? (
        <SearchUnavailable
          availability="not-configured"
          providerName={providerName}
          onLookup={() => void setMode("lookup")}
        />
      ) : mode === "search" ? (
        availability === "unsupported" ? (
          <SearchUnavailable
            availability="unsupported"
            providerName={providerName}
            onLookup={() => void setMode("lookup")}
          />
        ) : (
          <>
            <SourcingSearchForm
              initialValues={submitted?.values ?? null}
              autocompleteEnabled={
                availability === "unknown" || carrierIntelSupportsAutocomplete(provider)
              }
              isSearching={searchQuery.isFetching}
              onSearch={(values) => setSubmitted({ values, offset: 0 })}
              onReset={() => setSubmitted(null)}
              onPickSuggestion={openLookupFor}
            />
            <SourcingResults
              page={searchInput ? searchQuery.data : undefined}
              isLoading={searchQuery.isPending && searchInput !== null}
              isFetching={searchQuery.isFetching}
              error={searchQuery.isError ? searchQuery.error : null}
              offset={submitted?.offset ?? 0}
              pageSize={SOURCING_PAGE_SIZE}
              canImport={canImport}
              ruleLabels={ruleLabels}
              onImport={handleImport}
              onPageChange={(offset) =>
                setSubmitted((current) => (current ? { ...current, offset } : current))
              }
              onRetry={() => void searchQuery.refetch()}
            />
          </>
        )
      ) : (
        <ProspectLookup
          provider={provider}
          request={lookupRequest}
          prefill={lookupPrefill}
          canImport={canImport}
          ruleLabels={ruleLabels}
          onLookup={setLookupRequest}
          onImport={handleImport}
        />
      )}
      <ImportCarrierDialog
        candidate={importing}
        canEnrollMonitoring={provider?.configured ?? true}
        onOpenChange={(open) => {
          if (!open) {
            setImporting(null);
          }
        }}
      />
    </div>
  );
}
