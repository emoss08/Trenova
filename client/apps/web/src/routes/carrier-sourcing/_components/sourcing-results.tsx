import { useT } from "@trenova/shared/i18n/use-t";
import { ProviderErrorAlert } from "@/components/carrier-intelligence/provider-error-alert";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import type { CarrierSourcingPage } from "@/lib/graphql/carrier-sourcing";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatNumber } from "@trenova/shared/i18n/format";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { SourcingResultCard, type SourcingCandidate } from "./sourcing-result-card";

export type SourcingResultsProps = {
  page: CarrierSourcingPage | undefined;
  isLoading: boolean;
  isFetching: boolean;
  error: unknown;
  offset: number;
  pageSize: number;
  canImport: boolean;
  ruleLabels: Readonly<Record<string, string>>;
  onImport: (candidate: SourcingCandidate) => void;
  onPageChange: (offset: number) => void;
  onRetry: () => void;
};

export function SourcingResultsSketch() {
  return (
    <div className="flex flex-col gap-2">
      {[65, 40, 80].map((share) => (
        <div key={share} className="flex flex-col gap-2 rounded-lg border p-3">
          <div className="flex items-center gap-2">
            <GhostLine className="w-40" />
            <GhostLine className="w-16" />
          </div>
          <div className="grid grid-cols-4 gap-2">
            <GhostLine className="w-10" />
            <GhostLine className="w-10" />
            <GhostLine className="w-14" />
            <GhostLine className="w-12" />
          </div>
          <GhostBar share={share} className="w-full" />
        </div>
      ))}
    </div>
  );
}

export function SourcingResults({
  page,
  isLoading,
  isFetching,
  error,
  offset,
  pageSize,
  canImport,
  ruleLabels,
  onImport,
  onPageChange,
  onRetry,
}: SourcingResultsProps) {
  const t = useT();

  if (isLoading) {
    return (
      <div className="flex flex-col gap-3" aria-busy>
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <ProviderErrorAlert error={error} title={t("The search did not run")} onRetry={onRetry} />
    );
  }

  if (!page) {
    return (
      <EmptySheet
        title={t("Find carriers to add to your network")}
        description={t(
          "Search the provider's carrier registry by name, EIN, VIN or home state, and rank the results by the lanes you need covered. Every result is scored against your vetting rules before you import it.",
        )}
        sketch={<SourcingResultsSketch />}
      />
    );
  }

  const providerName = carrierIntelProviderLabel(page.provider);
  const hasPrevious = offset > 0;
  const hasNext = offset + pageSize < page.total;
  const rangeEnd = Math.min(offset + pageSize, page.total);

  return (
    <div className="flex flex-col gap-3" aria-busy={isFetching}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-muted-foreground text-xs" data-testid="sourcing-result-summary">
          {page.total === 0
            ? t("{0} found no carriers.", providerName)
            : t(
                "{0} matched {1} carriers. Showing {2}–{3}; {4} passed your filters on this page.",
                providerName,
                formatNumber(page.total),
                formatNumber(offset + 1),
                formatNumber(rangeEnd),
                formatNumber(page.items.length),
              )}
        </p>
        {hasPrevious || hasNext ? (
          <div className="flex items-center gap-1">
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!hasPrevious || isFetching}
              onClick={() => onPageChange(Math.max(offset - pageSize, 0))}
            >
              <ChevronLeftIcon className="size-3.5" />
              {t("Previous")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!hasNext || isFetching}
              isLoading={isFetching}
              onClick={() => onPageChange(offset + pageSize)}
            >
              {t("Next")}
              <ChevronRightIcon className="size-3.5" />
            </Button>
          </div>
        ) : null}
      </div>
      {page.items.length === 0 ? (
        <EmptySheet
          title={t("Nothing matches")}
          description={
            page.total > 0
              ? t(
                  "The provider returned carriers, but none on this page passed your power unit, authority age, hazmat or exclusion filters. Try the next page or loosen the filters.",
                )
              : t(
                  "No carrier fits this search. Try a shorter name, a different state, or fewer filters.",
                )
          }
          sketch={<SourcingResultsSketch />}
        />
      ) : (
        <ul className="flex flex-col gap-3">
          {page.items.map((item) => (
            <SourcingResultCard
              key={item.dotNumber}
              candidate={item}
              provider={page.provider}
              canImport={canImport}
              ruleLabels={ruleLabels}
              onImport={onImport}
            />
          ))}
        </ul>
      )}
    </div>
  );
}
