import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { EmptyState } from "@/components/empty-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { BankReceipt, BankReceiptStatus, MatchSuggestion } from "@/types/bank-receipt";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BanknoteIcon,
  CheckCircle2Icon,
  LinkIcon,
  ReceiptTextIcon,
  SearchIcon,
  TriangleAlertIcon,
  WalletCardsIcon,
} from "lucide-react";
import { parseAsString, useQueryStates } from "nuqs";
import { type ReactNode, useDeferredValue, useEffect, useMemo, useRef } from "react";
import { toast } from "sonner";

const bankReceiptSearchParams = {
  item: parseAsString,
  query: parseAsString.withDefault(""),
  status: parseAsString,
};

const STATUS_FILTERS: Array<{ label: string; value: BankReceiptStatus }> = [
  { label: "Imported", value: "Imported" },
  { label: "Matched", value: "Matched" },
  { label: "Exception", value: "Exception" },
];

export function BankReceiptPage() {
  const [searchParams, setSearchParams] = useQueryStates(bankReceiptSearchParams);
  const { item: selectedReceiptId, query, status } = searchParams;
  const deferredQuery = useDeferredValue(query);
  const queryClient = useQueryClient();
  const observerTarget = useRef<HTMLDivElement>(null);

  const {
    data: listData,
    isLoading,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useInfiniteQuery({
    queryKey: ["bankReceipt", "list", deferredQuery, status],
    queryFn: async ({ pageParam }) => {
      const params: Record<string, string> = {
        limit: "20",
        offset: String(pageParam),
      };
      if (deferredQuery.trim()) {
        params.query = deferredQuery.trim();
      }
      if (status) {
        params.status = status;
      }
      return apiService.bankReceiptService.getExceptions(params);
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.length === 20) {
        return lastPageParam + 20;
      }
      return undefined;
    },
  });

  const allRows = useMemo(
    () => listData?.pages.flatMap((page) => page) ?? [],
    [listData?.pages],
  );

  const selectedRow =
    allRows.find((row) => row.id === selectedReceiptId) ?? allRows[0] ?? null;

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const detailQuery = useQuery({
    ...queries.bankReceipt.get(selectedRow?.id ?? ""),
    enabled: Boolean(selectedRow?.id),
  });

  const summaryQuery = useQuery({
    ...queries.bankReceipt.summary(),
  });

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: "Bank Receipt Reconciliation",
        description:
          "Match imported bank receipts to customer payments and resolve exceptions.",
      }}
      toolbar={
        <div className="mx-4 mt-3 grid gap-2.5 md:grid-cols-4">
          <SummaryCard
            label="Imported"
            value={String(summaryQuery.data?.importedCount ?? 0)}
            amount={summaryQuery.data?.importedAmount}
          />
          <SummaryCard
            label="Matched"
            value={String(summaryQuery.data?.matchedCount ?? 0)}
            amount={summaryQuery.data?.matchedAmount}
          />
          <SummaryCard
            label="Exceptions"
            value={String(summaryQuery.data?.exceptionCount ?? 0)}
            amount={summaryQuery.data?.exceptionAmount}
          />
          <SummaryCard
            label="Active Work Items"
            value={String(summaryQuery.data?.activeWorkItemCount ?? 0)}
          />
        </div>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-1.5 border-b p-2">
            <Input
              value={query}
              onChange={(event) => void setSearchParams({ query: event.target.value })}
              placeholder="Search reference, memo..."
              leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
              className="h-7 text-xs"
            />
            <Select
              value={status ?? "all"}
              items={STATUS_FILTERS}
              onValueChange={(value) =>
                void setSearchParams({ status: value === "all" ? null : value })
              }
            >
              <SelectTrigger className="h-7 text-xs">
                <SelectValue placeholder="All statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                {STATUS_FILTERS.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <ScrollArea className="flex-1">
            <div
              className={cn(
                "flex flex-col gap-1.5 p-2",
                !isLoading && allRows.length === 0 && "h-full p-0",
              )}
            >
              {isLoading
                ? Array.from({ length: 6 }).map((_, index) => (
                    <Skeleton key={index} className="h-20 w-full rounded-lg" />
                  ))
                : null}
              {!isLoading && allRows.length === 0 ? (
                <div className="flex h-full items-center justify-center">
                  <EmptyState
                    title="No bank receipts"
                    description="Adjust the filters or import new bank receipts."
                    icons={[BanknoteIcon, ReceiptTextIcon, WalletCardsIcon]}
                    className="flex h-full max-w-none flex-col items-center justify-center rounded-none border-none p-6 shadow-none"
                  />
                </div>
              ) : null}
              {allRows.map((row) => {
                const isSelected = row.id === selectedRow?.id;
                return (
                  <button
                    key={row.id}
                    type="button"
                    onClick={() => void setSearchParams({ item: row.id })}
                    className={cn(
                      "rounded-lg border p-2.5 text-left transition-colors",
                      isSelected ? "border-primary bg-primary/5" : "hover:bg-muted/40",
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-xs font-medium">
                          {row.referenceNumber}
                        </p>
                        <p className="truncate text-2xs text-muted-foreground">
                          {formatReceiptDate(row.receiptDate)}
                        </p>
                      </div>
                      <AmountDisplay
                        value={row.amountMinor}
                        className="shrink-0 text-xs font-semibold"
                      />
                    </div>
                    <div className="mt-1.5 flex items-center gap-1.5">
                      <AccountingStatusBadge status={row.status} />
                    </div>
                    {row.memo ? (
                      <p className="mt-1.5 line-clamp-1 text-2xs text-muted-foreground">
                        {row.memo}
                      </p>
                    ) : null}
                  </button>
                );
              })}
              {isFetchingNextPage ? (
                <div className="flex items-center justify-center py-4">
                  <TextShimmer className="font-mono text-sm" duration={1}>
                    Loading more...
                  </TextShimmer>
                </div>
              ) : null}
              <div ref={observerTarget} className="h-px" />
            </div>
          </ScrollArea>
        </div>
      }
      detail={
        <ScrollArea className="h-full">
          {!selectedRow ? (
            <div className="flex h-full items-center justify-center p-6">
              <EmptyState
                title="No receipt selected"
                description="Select a bank receipt from the list to view details and match suggestions."
                icons={[BanknoteIcon, ReceiptTextIcon, WalletCardsIcon]}
                className="max-w-xl border-none p-8 shadow-none"
              />
            </div>
          ) : detailQuery.isLoading || !detailQuery.data ? (
            <div className="space-y-4 p-4">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
          ) : (
            <ReceiptDetail
              receipt={detailQuery.data}
              queryClient={queryClient}
            />
          )}
        </ScrollArea>
      }
    />
  );
}

function ReceiptDetail({
  receipt,
  queryClient,
}: {
  receipt: BankReceipt;
  queryClient: ReturnType<typeof useQueryClient>;
}) {
  const suggestionsQuery = useQuery({
    ...queries.bankReceipt.suggestions(receipt.id!),
    enabled: receipt.status !== "Matched" && Boolean(receipt.id),
  });

  const matchMutation = useMutation({
    mutationFn: async (customerPaymentId: string) =>
      apiService.bankReceiptService.match(receipt.id!, customerPaymentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["bankReceipt"] });
      toast.success("Receipt matched to payment");
    },
    onError: () => toast.error("Failed to match receipt"),
  });

  return (
    <div className="flex flex-col gap-5 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{receipt.referenceNumber}</h2>
          <p className="text-sm text-muted-foreground">
            {formatReceiptDate(receipt.receiptDate)}
          </p>
        </div>
        <AmountDisplay
          value={receipt.amountMinor}
          className="text-2xl font-bold"
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <AccountingStatusBadge status={receipt.status} />
      </div>

      <div className="grid gap-5 xl:grid-cols-2">
        <div className="flex flex-col gap-5">
          <div className="rounded-lg border bg-card p-3">
            <SectionLabel>Receipt Details</SectionLabel>
            <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
              <PropertyCell label="Reference">
                <span className="text-xs font-medium">{receipt.referenceNumber}</span>
              </PropertyCell>
              <PropertyCell label="Date">
                <span className="text-xs font-medium">
                  {formatReceiptDate(receipt.receiptDate)}
                </span>
              </PropertyCell>
              <PropertyCell label="Amount">
                <AmountDisplay value={receipt.amountMinor} className="text-xs font-medium" />
              </PropertyCell>
              <PropertyCell label="Status">
                <AccountingStatusBadge status={receipt.status} />
              </PropertyCell>
            </div>
            {receipt.memo ? (
              <div className="mt-2.5">
                <PropertyCell label="Memo">
                  <p className="mt-0.5 text-xs text-muted-foreground">{receipt.memo}</p>
                </PropertyCell>
              </div>
            ) : null}
          </div>

          {receipt.status === "Exception" && receipt.exceptionReason ? (
            <div className="rounded-lg border border-red-200 bg-red-50/50 p-3 dark:border-red-900/50 dark:bg-red-950/20">
              <div className="flex items-center gap-1.5">
                <TriangleAlertIcon className="size-3.5 text-red-600 dark:text-red-400" />
                <SectionLabel>Exception Reason</SectionLabel>
              </div>
              <p className="mt-1.5 text-xs text-red-700 dark:text-red-300">
                {receipt.exceptionReason}
              </p>
            </div>
          ) : null}

          {receipt.status === "Matched" ? (
            <div className="rounded-lg border border-green-200 bg-green-50/50 p-3 dark:border-green-900/50 dark:bg-green-950/20">
              <div className="flex items-center gap-1.5">
                <CheckCircle2Icon className="size-3.5 text-green-600 dark:text-green-400" />
                <SectionLabel>Matched Payment</SectionLabel>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
                <PropertyCell label="Payment ID">
                  <span className="font-mono text-xs font-medium">
                    {receipt.matchedCustomerPaymentId ?? "N/A"}
                  </span>
                </PropertyCell>
                <PropertyCell label="Matched At">
                  <span className="text-xs font-medium">
                    {receipt.matchedAt
                      ? new Date(receipt.matchedAt * 1000).toLocaleString()
                      : "N/A"}
                  </span>
                </PropertyCell>
              </div>
            </div>
          ) : null}
        </div>

        <div className="flex flex-col gap-5">
          {receipt.status !== "Matched" ? (
            <div className="rounded-lg border bg-card p-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <LinkIcon className="size-3.5 text-muted-foreground" />
                  <SectionLabel>Match Suggestions</SectionLabel>
                </div>
                {suggestionsQuery.data ? (
                  <Badge variant="secondary">
                    {suggestionsQuery.data.length} found
                  </Badge>
                ) : null}
              </div>
              <div className="mt-2">
                {suggestionsQuery.isLoading ? (
                  <div className="space-y-2">
                    <Skeleton className="h-10 w-full" />
                    <Skeleton className="h-10 w-full" />
                    <Skeleton className="h-10 w-full" />
                  </div>
                ) : !suggestionsQuery.data || suggestionsQuery.data.length === 0 ? (
                  <p className="py-4 text-center text-xs text-muted-foreground">
                    No match suggestions available.
                  </p>
                ) : (
                  <SuggestionTable
                    suggestions={suggestionsQuery.data}
                    onMatch={(customerPaymentId) => matchMutation.mutate(customerPaymentId)}
                    isMatching={matchMutation.isPending}
                  />
                )}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function SuggestionTable({
  suggestions,
  onMatch,
  isMatching,
}: {
  suggestions: MatchSuggestion[];
  onMatch: (customerPaymentId: string) => void;
  isMatching: boolean;
}) {
  return (
    <div className="overflow-hidden rounded-md border">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-left text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-xs font-medium">Reference</th>
            <th className="px-3 py-2 text-right text-xs font-medium">Amount</th>
            <th className="px-3 py-2 text-right text-xs font-medium">Score</th>
            <th className="px-3 py-2 text-xs font-medium">Reason</th>
            <th className="px-3 py-2 text-right text-xs font-medium">Action</th>
          </tr>
        </thead>
        <tbody>
          {suggestions.map((suggestion) => (
            <tr
              key={suggestion.customerPaymentId}
              className="border-t transition-colors hover:bg-muted/50"
            >
              <td className="px-3 py-2 font-mono text-2xs">
                {suggestion.referenceNumber}
              </td>
              <td className="px-3 py-2 text-right text-xs tabular-nums">
                <AmountDisplay value={suggestion.amountMinor} className="text-xs" />
              </td>
              <td className="px-3 py-2 text-right">
                <ScoreBadge score={suggestion.score} />
              </td>
              <td className="max-w-[200px] px-3 py-2 text-2xs text-muted-foreground">
                <span className="line-clamp-2">{suggestion.reason}</span>
              </td>
              <td className="px-3 py-2 text-right">
                <Button
                  size="xs"
                  type="button"
                  className="bg-green-600 text-white hover:bg-green-700"
                  onClick={() => onMatch(suggestion.customerPaymentId)}
                  disabled={isMatching}
                >
                  Match
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ScoreBadge({ score }: { score: number }) {
  const variant = score >= 80 ? "active" : score >= 50 ? "orange" : "secondary";
  return (
    <Badge variant={variant} className="tabular-nums">
      {score}%
    </Badge>
  );
}

function SummaryCard({
  label,
  value,
  amount,
}: {
  label: string;
  value: string;
  amount?: number;
}) {
  return (
    <div className="rounded-lg border bg-card px-3 py-2.5">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
      {amount !== undefined ? (
        <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
          {formatCurrency(amount / 100)}
        </p>
      ) : null}
    </div>
  );
}

function SectionLabel({ children }: { children: ReactNode }) {
  return <p className="text-xs font-medium text-muted-foreground">{children}</p>;
}

function PropertyCell({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <p className="text-2xs text-muted-foreground">{label}</p>
      {children}
    </div>
  );
}

function formatReceiptDate(epoch: number) {
  return new Date(epoch * 1000).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
