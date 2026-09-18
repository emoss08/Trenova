import { BillingListEmpty } from "@/components/billing/billing-empty";
import { useInfiniteScrollSentinel } from "@/hooks/use-infinite-scroll-sentinel";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateFilters,
  BillingTransferCandidateStatus,
} from "@/lib/graphql/billing-transfer";
import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { SearchIcon } from "lucide-react";

type StatusFilterValue = "all" | BillingTransferCandidateStatus;

const STATUS_FILTERS: readonly { value: StatusFilterValue; label: string }[] = [
  { value: "all", label: "All eligible" },
  { value: "ReadyToInvoice", label: "Ready to Invoice" },
  { value: "Completed", label: "Completed" },
];

const LOADING_ROWS = 6;

type BulkBillingTransferCandidatesProps = {
  filters: BillingTransferCandidateFilters;
  onFiltersChange: (filters: BillingTransferCandidateFilters) => void;
  candidates: BillingTransferCandidate[];
  totalCount: number | null;
  isLoading: boolean;
  isError: boolean;
  onRetryLoad: () => void;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
  selectedIds: ReadonlySet<string>;
  onSelectionChange: (candidates: BillingTransferCandidate[], selected: boolean) => void;
};

export function BulkBillingTransferCandidates({
  filters,
  onFiltersChange,
  candidates,
  totalCount,
  isLoading,
  isError,
  onRetryLoad,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  selectedIds,
  onSelectionChange,
}: BulkBillingTransferCandidatesProps) {
  const t = useT();

  const isFiltered = filters.query.trim() !== "" || filters.status !== null;
  const selectedShown = candidates.reduce(
    (count, candidate) => (selectedIds.has(candidate.id) ? count + 1 : count),
    0,
  );
  const allShownSelected = candidates.length > 0 && selectedShown === candidates.length;
  const sentinelRef = useInfiniteScrollSentinel<HTMLLIElement>({
    hasNextPage,
    isFetchingNextPage,
    onLoadMore,
  });

  return (
    <div className="flex min-h-0 flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          placeholder={t("Search PRO, BOL...")}
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          value={filters.query}
          onChange={(event) => onFiltersChange({ ...filters, query: event.target.value })}
          className="h-8 text-xs"
          inputContainerClassName="min-w-48 flex-1"
        />
        <SegmentedControl
          aria-label={t("Shipment status")}
          items={STATUS_FILTERS.map((item) => ({ value: item.value, label: t(item.label) }))}
          value={filters.status ?? "all"}
          onValueChange={(value) =>
            onFiltersChange({ ...filters, status: value === "all" ? null : value })
          }
        />
      </div>

      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          {totalCount === null
            ? t("Counting shipments...")
            : t(
                "{0, plural, one {# shipment can transfer} other {# shipments can transfer}}",
                totalCount,
              )}
        </span>
        {totalCount !== null && candidates.length > 0 && candidates.length < totalCount ? (
          <span className="tabular-nums">
            {t("Showing {0} of {1}", candidates.length, totalCount)}
          </span>
        ) : null}
      </div>

      <div className="overflow-hidden rounded-lg border">
        <div className="bg-muted/40 text-muted-foreground grid grid-cols-[1.5rem_minmax(0,1.2fr)_minmax(0,1.4fr)_7.5rem_6rem_6rem] items-center gap-3 border-b px-3 py-2 text-xs font-medium">
          <Checkbox
            aria-label={t("Select all shown shipments")}
            checked={allShownSelected}
            indeterminate={selectedShown > 0 && !allShownSelected}
            disabled={candidates.length === 0}
            onCheckedChange={(checked) => onSelectionChange(candidates, checked === true)}
          />
          <span>{t("PRO")}</span>
          <span>{t("Customer")}</span>
          <span>{t("Status")}</span>
          <span>{t("Delivered")}</span>
          <span className="text-right">{t("Total")}</span>
        </div>

        <ScrollArea className="h-[min(26rem,50vh)]">
          {isLoading ? (
            <div className="flex flex-col gap-2 p-3" aria-busy="true">
              {Array.from({ length: LOADING_ROWS }, (_, index) => (
                <Skeleton key={index} className="h-8 w-full" />
              ))}
            </div>
          ) : isError ? (
            <div className="flex flex-col items-center gap-3 px-6 py-10 text-center">
              <p className="text-sm font-medium">{t("The shipments could not be loaded")}</p>
              <Button variant="outline" size="sm" onClick={onRetryLoad}>
                {t("Try again")}
              </Button>
            </div>
          ) : candidates.length === 0 ? (
            <BillingListEmpty
              title={isFiltered ? t("Nothing matches") : t("Nothing to transfer")}
              description={
                isFiltered
                  ? t(
                      "No shipment that can transfer fits the search and status. Widen them, or clear them to see every eligible shipment.",
                    )
                  : t(
                      "Every Completed and Ready to Invoice shipment is already in the billing queue.",
                    )
              }
              onClearFilters={
                isFiltered ? () => onFiltersChange({ query: "", status: null }) : undefined
              }
            />
          ) : (
            <ul aria-label={t("Shipments that can transfer")} className="divide-y">
              {candidates.map((candidate) => (
                <CandidateRow
                  key={candidate.id}
                  candidate={candidate}
                  selected={selectedIds.has(candidate.id)}
                  onSelectedChange={(selected) => onSelectionChange([candidate], selected)}
                />
              ))}
              {hasNextPage ? (
                <li ref={sentinelRef} className="flex justify-center p-2" aria-live="polite">
                  {isFetchingNextPage ? (
                    <span className="text-muted-foreground flex items-center gap-2 text-xs">
                      <Spinner className="size-3.5" />
                      {t("Loading more shipments...")}
                    </span>
                  ) : (
                    <Button variant="ghost" size="sm" onClick={onLoadMore}>
                      {t("Load more")}
                    </Button>
                  )}
                </li>
              ) : null}
            </ul>
          )}
        </ScrollArea>
      </div>
    </div>
  );
}

function CandidateRow({
  candidate,
  selected,
  onSelectedChange,
}: {
  candidate: BillingTransferCandidate;
  selected: boolean;
  onSelectedChange: (selected: boolean) => void;
}) {
  const t = useT();

  return (
    <li
      className={cn(
        "grid grid-cols-[1.5rem_minmax(0,1.2fr)_minmax(0,1.4fr)_7.5rem_6rem_6rem] items-center gap-3 px-3 py-2 text-xs",
        selected && "bg-brand/5",
      )}
    >
      <Checkbox
        aria-label={t("Select {0}", candidate.proNumber)}
        checked={selected}
        onCheckedChange={(checked) => onSelectedChange(checked === true)}
      />
      <div className="flex min-w-0 flex-col">
        <span className="truncate font-mono font-medium">{candidate.proNumber}</span>
        {candidate.bol ? (
          <span className="text-muted-foreground truncate text-xs">
            {t("BOL {0}", candidate.bol)}
          </span>
        ) : null}
      </div>
      <span className={cn("truncate", !candidate.customer && "text-muted-foreground")}>
        {candidate.customer?.name ?? t("No customer")}
      </span>
      <div className="flex min-w-0 flex-col items-start gap-1">
        <ShipmentStatusBadge status={candidate.status} />
        {candidate.billingTransferStatus === "SentBackToOps" ? (
          <Badge variant="warning" className="max-h-5 text-2xs">
            {t("Sent back")}
          </Badge>
        ) : null}
      </div>
      <span className="text-muted-foreground tabular-nums">
        {formatUnixInUserTimezone(
          candidate.actualDeliveryDate,
          { month: "short", day: "numeric", year: "numeric" },
          "—",
        )}
      </span>
      <span className="text-right tabular-nums">
        {formatCurrency(Number(candidate.totalChargeAmount))}
      </span>
    </li>
  );
}
