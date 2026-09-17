import { useT } from "@trenova/shared/i18n/use-t";
import {
  CarrierIntelProfileView,
  type CarrierIntelProfileSectionId,
} from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { CarrierKeyFacts } from "@/components/carrier-intelligence/carrier-key-facts";
import { DecisionSummary } from "@/components/carrier-intelligence/decision-summary";
import { FreshnessIndicator } from "@/components/carrier-intelligence/freshness-indicator";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { RiskLabel } from "@/components/carrier-intelligence/status-dot";
import {
  availableVetDepths,
  carrierIntelProviderLabel,
  carrierIntelVetCost,
  type CarrierIntelVetCost,
} from "@/lib/carrier-intelligence";
import { carrierPanelPath } from "@/lib/carrier-links";
import {
  candidateFromLookup,
  withImportedCarriers,
  type CarrierIntelProviderCapabilities,
  type SourcingCandidate,
} from "@/lib/carrier-sourcing";
import {
  CARRIER_INTEL_LOOKUP_KEY,
  lookupCarrierIntelProspect,
} from "@/lib/graphql/carrier-sourcing";
import type {
  CarrierIntelDepth,
  CarrierIntelLookupInput,
} from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { ArrowUpRightIcon, LayersIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { candidateMetaLine } from "./sourcing-result-list";

const SOURCING_PROFILE_SECTIONS = [
  "authority",
  "insurance",
  "safety",
  "network",
  "lanes",
] as const satisfies readonly CarrierIntelProfileSectionId[];

export type CarrierSheetTarget = {
  dotNumber: string;
  seed: SourcingCandidate | null;
};

export type CarrierDetailSheetProps = {
  target: CarrierSheetTarget | null;
  provider: CarrierIntelProviderCapabilities | null;
  imported: Readonly<Record<string, string>>;
  canImport: boolean;
  ruleLabels: Readonly<Record<string, string>>;
  onOpenChange: (open: boolean) => void;
  onImport: (candidate: SourcingCandidate) => void;
};

export function previewDepth(
  provider: CarrierIntelProviderCapabilities | null,
): CarrierIntelDepth | null {
  if (!provider?.configured) {
    return null;
  }
  const depths = availableVetDepths(provider.capabilities);
  if (depths.includes("Lite")) {
    return "Lite";
  }
  return depths[0] ?? null;
}

export function lookupQueryOptions(input: CarrierIntelLookupInput, enabled: boolean) {
  return {
    queryKey: [CARRIER_INTEL_LOOKUP_KEY, input] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => lookupCarrierIntelProspect(input, { signal }),
    enabled,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
    refetchOnWindowFocus: false,
  };
}

function useCostLabel() {
  const t = useT();
  return (cost: CarrierIntelVetCost): string | null => {
    switch (cost.basis) {
      case "PerDOTMonth":
        return t("~{0}/mo", formatCurrency(cost.amount));
      case "PerMatch":
        return t("~{0} per lookup", formatCurrency(cost.amount));
      default:
        return null;
    }
  };
}

function SheetSkeleton() {
  return (
    <div className="flex flex-col gap-6 p-5" aria-busy>
      <div className="flex flex-col gap-2">
        <Skeleton className="h-5 w-64" />
        <Skeleton className="h-3.5 w-80" />
        <Skeleton className="h-3.5 w-40" />
      </div>
      <div className="flex flex-col gap-2">
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
      <div className="grid grid-cols-2 gap-4">
        {Array.from({ length: 8 }, (_, index) => (
          <div key={index} className="flex flex-col gap-1.5">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-4 w-32" />
          </div>
        ))}
      </div>
    </div>
  );
}

function CarrierSheetBody({
  target,
  provider,
  imported,
  canImport,
  ruleLabels,
  onImport,
}: Omit<CarrierDetailSheetProps, "target" | "onOpenChange"> & { target: CarrierSheetTarget }) {
  const t = useT();
  const costLabel = useCostLabel();
  const [fullRequested, setFullRequested] = useState(false);
  const depth = previewDepth(provider);
  const supportsFull = provider?.configured
    ? availableVetDepths(provider.capabilities).includes("Full")
    : false;

  const previewQuery = useQuery(
    lookupQueryOptions({ dotNumber: target.dotNumber, depth }, target.seed === null),
  );
  const fullQuery = useQuery(
    lookupQueryOptions({ dotNumber: target.dotNumber, depth: "Full" }, fullRequested),
  );

  const base =
    fullQuery.data !== undefined
      ? candidateFromLookup(fullQuery.data)
      : (target.seed ?? (previewQuery.data ? candidateFromLookup(previewQuery.data) : null));
  const candidate = base ? withImportedCarriers(base, imported) : null;

  if (!candidate) {
    return (
      <>
        <SheetTitle className="sr-only">{t("USDOT {0}", target.dotNumber)}</SheetTitle>
        <SheetDescription className="sr-only">{t("Loading carrier")}</SheetDescription>
        {previewQuery.isError ? (
          <div className="p-5">
            <IntelInlineError
              error={previewQuery.error}
              title={t("Couldn't pull USDOT {0}", target.dotNumber)}
              onRetry={() => void previewQuery.refetch()}
            />
          </div>
        ) : (
          <SheetSkeleton />
        )}
      </>
    );
  }

  const identity = candidate.profile.identity;
  const name = candidate.legalName || identity?.legalName || t("USDOT {0}", candidate.dotNumber);
  const fullCost = carrierIntelVetCost(provider?.provider, "Full");
  const fullCostLabel = costLabel(fullCost);
  const showPullFull = supportsFull && candidate.depth !== "Full" && !candidate.notFound;

  return (
    <>
      <header className="flex flex-col gap-1 border-b px-5 pt-5 pb-4 pr-12">
        <SheetTitle className="truncate text-base font-medium">{name}</SheetTitle>
        {identity?.dbaName && identity.dbaName !== name ? (
          <p className="text-muted-foreground truncate text-sm">{identity.dbaName}</p>
        ) : null}
        <SheetDescription className="text-muted-foreground truncate text-xs tabular-nums">
          {candidateMetaLine(candidate, t)}
        </SheetDescription>
        <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1">
          <RiskLabel level={candidate.riskLevel} />
          {candidate.asOf ? (
            <FreshnessIndicator
              effectiveAsOf={candidate.asOf}
              fetchedAt={candidate.fetchedAt}
              sourceAsOf={candidate.sourceAsOf}
              depth={candidate.depth}
              depthFetchedAt={candidate.depthFetchedAt}
              fetchedDepth={candidate.fetchedDepth}
              provider={candidate.provider}
              assessStaleness={false}
            />
          ) : null}
        </div>
      </header>
      <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-5 py-5">
        <DecisionSummary
          findings={candidate.findings}
          ruleLabels={ruleLabels}
          notFound={candidate.notFound}
          provider={candidate.provider}
        />
        {!candidate.notFound ? (
          <>
            <CarrierKeyFacts profile={candidate.profile} />
            {showPullFull ? (
              <div className="flex items-center justify-between gap-3 border-y py-2">
                <span className="text-muted-foreground text-xs">
                  {fullQuery.isError
                    ? t("The full profile didn't load. Try again.")
                    : t("Network signals and lanes come with the full profile.")}
                </span>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        isLoading={fullQuery.isFetching}
                        onClick={() => {
                          if (fullRequested) {
                            void fullQuery.refetch();
                          } else {
                            setFullRequested(true);
                          }
                        }}
                      />
                    }
                  >
                    <LayersIcon className="size-3.5" aria-hidden />
                    {fullCostLabel
                      ? t("Pull full profile · {0}", fullCostLabel)
                      : t("Pull full profile")}
                  </TooltipTrigger>
                  <TooltipContent className="max-w-64">
                    {fullCost.basis === "PerDOTMonth"
                      ? t(
                          "{0} bills about {1} per carrier per month at full depth. Pulling it again this month costs nothing extra.",
                          carrierIntelProviderLabel(provider?.provider),
                          formatCurrency(fullCost.amount),
                        )
                      : fullCost.basis === "PerMatch"
                        ? t(
                            "{0} bills about {1} for each matched lookup.",
                            carrierIntelProviderLabel(provider?.provider),
                            formatCurrency(fullCost.amount),
                          )
                        : t(
                            "{0} doesn't charge per lookup.",
                            carrierIntelProviderLabel(provider?.provider),
                          )}
                  </TooltipContent>
                </Tooltip>
              </div>
            ) : null}
            <CarrierIntelProfileView
              profile={candidate.profile}
              provider={candidate.provider}
              sections={SOURCING_PROFILE_SECTIONS}
            />
          </>
        ) : null}
      </div>
      <footer className="bg-background flex items-center justify-end gap-2 border-t px-5 py-3">
        {candidate.existingCarrierId ? (
          <Button
            nativeButton={false}
            render={<Link to={carrierPanelPath(candidate.existingCarrierId, "intelligence")} />}
          >
            {t("Open in Trenova")}
            <ArrowUpRightIcon className="size-3.5" aria-hidden />
          </Button>
        ) : candidate.notFound ? null : canImport ? (
          <Button type="button" onClick={() => onImport(candidate)}>
            {t("Import carrier")}
          </Button>
        ) : (
          <span className="text-muted-foreground text-xs">
            {t("You don't have permission to import carriers.")}
          </span>
        )}
      </footer>
    </>
  );
}

export function CarrierDetailSheet({ target, onOpenChange, ...rest }: CarrierDetailSheetProps) {
  const [shown, setShown] = useState<CarrierSheetTarget | null>(target);
  if (target !== null && target !== shown) {
    setShown(target);
  }

  return (
    <Sheet open={target !== null} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="gap-0 overflow-hidden p-0 data-[side=right]:w-[calc(100%-2rem)] data-[side=right]:sm:max-w-[600px]"
      >
        {shown ? <CarrierSheetBody key={shown.dotNumber} target={shown} {...rest} /> : null}
      </SheetContent>
    </Sheet>
  );
}
