import { useT } from "@trenova/shared/i18n/use-t";
import { BillingDetailUnselected, BillingListEmpty } from "@/components/billing/billing-empty";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";

import { resolutionTypeChoices, workItemStatusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  BankReceiptWorkItem,
  ResolutionType,
  WorkItemStatus,
} from "@/types/bank-receipt-work-item";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { PlayIcon, SearchIcon, ShieldCheckIcon, UserPlusIcon } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

const STATUS_LABELS: Record<WorkItemStatus, string> = {
  Open: "Open",
  Assigned: "Assigned",
  InReview: "In Review",
  Resolved: "Resolved",
  Dismissed: "Dismissed",
};

const STATUS_VARIANTS: Record<
  WorkItemStatus,
  "default" | "secondary" | "warning" | "info" | "active"
> = {
  Open: "secondary",
  Assigned: "info",
  InReview: "warning",
  Resolved: "active",
  Dismissed: "default",
};

export function BankReceiptQueuePage() {
  const t = useT();

  const [selectedWorkItemId, setSelectedWorkItemId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const hasActiveFilters = Boolean(searchQuery || statusFilter);
  const clearFilters = () => {
    setSearchQuery("");
    setStatusFilter(null);
  };
  const queryClient = useQueryClient();
  const observerTarget = useRef<HTMLDivElement>(null);

  const {
    data: listData,
    isLoading,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useInfiniteQuery({
    queryKey: ["bankReceiptWorkItems", searchQuery, statusFilter],
    queryFn: async ({ pageParam }) => {
      const params: Record<string, string> = {
        limit: "20",
        offset: String(pageParam),
      };
      if (searchQuery.trim()) {
        params.query = searchQuery.trim();
      }
      if (statusFilter) {
        params.fieldFilters = JSON.stringify([
          { field: "status", operator: "eq", value: statusFilter },
        ]);
      }
      return apiService.bankReceiptWorkItemService.list(params);
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.length === 20) {
        return lastPageParam + 20;
      }
      return undefined;
    },
  });

  const allRows = useMemo(() => listData?.pages.flat() ?? [], [listData?.pages]);

  const selectedRow = allRows.find((row) => row.id === selectedWorkItemId) ?? null;

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
    ...queries.bankReceiptWorkItem.get(selectedRow?.id ?? ""),
    enabled: Boolean(selectedRow?.id),
  });

  const summaryQuery = useQuery({
    ...queries.bankReceipt.summary(),
  });

  const invalidateAll = () => {
    void queryClient.invalidateQueries({ queryKey: ["bankReceiptWorkItems"] });
    void queryClient.invalidateQueries({ queryKey: ["bankReceiptWorkItem"] });
    void queryClient.invalidateQueries({ queryKey: ["bankReceipt"] });
  };

  const assignMutation = useMutation({
    mutationFn: async ({ id, userId }: { id: string; userId: string }) =>
      apiService.bankReceiptWorkItemService.assign(id, userId),
    onSuccess: () => {
      invalidateAll();
      toast.success(t("Work item assigned"));
    },
    onError: () => toast.error(t("Failed to assign work item")),
  });

  const startReviewMutation = useMutation({
    mutationFn: async (id: string) => apiService.bankReceiptWorkItemService.startReview(id),
    onSuccess: () => {
      invalidateAll();
      toast.success(t("Review started"));
    },
    onError: () => toast.error(t("Failed to start review")),
  });

  const resolveMutation = useMutation({
    mutationFn: async ({
      id,
      resolutionType,
      resolutionNote,
    }: {
      id: string;
      resolutionType: string;
      resolutionNote: string;
    }) => apiService.bankReceiptWorkItemService.resolve(id, { resolutionType, resolutionNote }),
    onSuccess: () => {
      invalidateAll();
      toast.success(t("Work item resolved"));
    },
    onError: () => toast.error(t("Failed to resolve work item")),
  });

  const dismissMutation = useMutation({
    mutationFn: async ({ id, resolutionNote }: { id: string; resolutionNote: string }) =>
      apiService.bankReceiptWorkItemService.dismiss(id, { resolutionNote }),
    onSuccess: () => {
      invalidateAll();
      toast.success(t("Work item dismissed"));
    },
    onError: () => toast.error(t("Failed to dismiss work item")),
  });

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: "Bank Receipt Work Queue",
        description: "Review and resolve bank receipt exceptions requiring attention.",
      }}
      className="p-0 gap-y-2"
      toolbar={
        <div className="mx-4 mt-3 grid gap-2.5 md:grid-cols-4">
          <SummaryCard
            label={t("Open")}
            value={String(
              (summaryQuery.data?.activeWorkItemCount ?? 0) -
                (summaryQuery.data?.assignedWorkItemCount ?? 0) -
                (summaryQuery.data?.inReviewWorkItemCount ?? 0),
            )}
          />
          <SummaryCard
            label={t("Assigned")}
            value={String(summaryQuery.data?.assignedWorkItemCount ?? 0)}
          />
          <SummaryCard
            label={t("In Review")}
            value={String(summaryQuery.data?.inReviewWorkItemCount ?? 0)}
          />
          <SummaryCard label={t("Exceptions")} value={String(summaryQuery.data?.exceptionCount ?? 0)} />
        </div>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-1.5 border-b p-2">
            <Input
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder={t("Search reference, ID...")}
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
              className="h-7 text-xs"
            />
            <Select
              value={statusFilter ?? "all"}
              onValueChange={(value) => setStatusFilter(value === "all" ? null : value)}
            >
              <SelectTrigger className="h-7 text-xs">
                <SelectValue placeholder={t("All statuses")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("All statuses")}</SelectItem>
                {workItemStatusChoices.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {t(choice.label)}
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
                <BillingListEmpty
                  title={hasActiveFilters ? "Nothing matches" : "Nothing waiting"}
                  description={
                    hasActiveFilters
                      ? "No work item fits the search and filters. Widen them, or clear them to see everything waiting."
                      : "A work item is raised when an imported receipt cannot be matched on its own. Until one is, there is nothing to resolve."
                  }
                  onClearFilters={hasActiveFilters ? clearFilters : undefined}
                />
              ) : null}
              {allRows.map((row) => {
                const isSelected = row.id === selectedWorkItemId;
                return (
                  <button
                    key={row.id}
                    type="button"
                    onClick={() => setSelectedWorkItemId(row.id ?? null)}
                    className={cn(
                      "rounded-lg border p-2.5 text-left transition-colors",
                      isSelected ? "border-primary bg-primary/5" : "hover:bg-muted/40",
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <p className="min-w-0 truncate text-xs font-medium">{row.bankReceiptId}</p>
                      <Badge variant={STATUS_VARIANTS[row.status]}>
                        {STATUS_LABELS[row.status]}
                      </Badge>
                    </div>
                    <div className="mt-1.5 flex items-center gap-1.5">
                      {row.assignedToUserId ? (
                        <span className="text-2xs text-muted-foreground">{t("Assigned")}</span>
                      ) : (
                        <span className="text-2xs text-muted-foreground">{t("Unassigned")}</span>
                      )}
                    </div>
                  </button>
                );
              })}
              {isFetchingNextPage ? (
                <div className="flex items-center justify-center py-4">
                  <TextShimmer className="font-mono text-sm" duration={1}>
                    {t("Loading more...")}
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
            <BillingDetailUnselected
              layout="cards"
              title={t("Nothing open")}
              description={t("Pick a work item from the list to review the receipt behind it and settle it.")}
            />
          ) : detailQuery.isLoading || !detailQuery.data ? (
            <div className="space-y-4 p-4">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-64 w-full" />
            </div>
          ) : (
            <WorkItemDetail
              workItem={detailQuery.data}
              assignMutation={assignMutation}
              startReviewMutation={startReviewMutation}
              resolveMutation={resolveMutation}
              dismissMutation={dismissMutation}
            />
          )}
        </ScrollArea>
      }
    />
  );
}

function WorkItemDetail({
  workItem,
  assignMutation,
  startReviewMutation,
  resolveMutation,
  dismissMutation,
}: {
  workItem: BankReceiptWorkItem;
  assignMutation: ReturnType<
    typeof useMutation<BankReceiptWorkItem, Error, { id: string; userId: string }>
  >;
  startReviewMutation: ReturnType<typeof useMutation<BankReceiptWorkItem, Error, string>>;
  resolveMutation: ReturnType<
    typeof useMutation<
      BankReceiptWorkItem,
      Error,
      { id: string; resolutionType: string; resolutionNote: string }
    >
  >;
  dismissMutation: ReturnType<
    typeof useMutation<BankReceiptWorkItem, Error, { id: string; resolutionNote: string }>
  >;
}) {
  const t = useT();

  const [resolutionType, setResolutionType] = useState<ResolutionType | "">("");
  const [resolutionNote, setResolutionNote] = useState("");
  const [showResolveForm, setShowResolveForm] = useState(false);
  const [showDismissForm, setShowDismissForm] = useState(false);

  const bankReceiptQuery = useQuery({
    ...queries.bankReceipt.get(workItem.bankReceiptId),
    enabled: Boolean(workItem.bankReceiptId),
  });

  const receipt = bankReceiptQuery.data;

  return (
    <div className="flex flex-col gap-5 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{workItem.id}</h2>
          <p className="text-muted-foreground text-sm">{t("Bank Receipt: {0}", workItem.bankReceiptId)}</p>
        </div>
        <Badge variant={STATUS_VARIANTS[workItem.status]}>{STATUS_LABELS[workItem.status]}</Badge>
      </div>

      {receipt ? (
        <div className="bg-card rounded-lg border p-3">
          <SectionLabel>{t("Bank Receipt Info")}</SectionLabel>
          <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
            <PropertyCell label={t("Receipt Date")}>
              <span className="text-xs font-medium">{formatUnixDate(receipt.receiptDate)}</span>
            </PropertyCell>
            <PropertyCell label={t("Amount")}>
              <span className="text-xs font-medium tabular-nums">
                {formatCurrency(receipt.amountMinor / 100)}
              </span>
            </PropertyCell>
            <PropertyCell label={t("Reference")}>
              <span className="text-xs font-medium">{receipt.referenceNumber}</span>
            </PropertyCell>
            <PropertyCell label={t("Memo")}>
              <span className="text-xs font-medium">{receipt.memo || "—"}</span>
            </PropertyCell>
            <PropertyCell label={t("Status")}>
              <Badge variant="secondary">{receipt.status}</Badge>
            </PropertyCell>
            {receipt.exceptionReason ? (
              <PropertyCell label={t("Exception Reason")}>
                <span className="text-xs font-medium text-red-600 dark:text-red-400">
                  {receipt.exceptionReason}
                </span>
              </PropertyCell>
            ) : null}
          </div>
        </div>
      ) : bankReceiptQuery.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : null}

      {workItem.assignedToUserId ? (
        <div className="bg-card rounded-lg border p-3">
          <SectionLabel>{t("Assignment")}</SectionLabel>
          <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
            <PropertyCell label={t("Assigned To")}>
              <span className="text-xs font-medium">{workItem.assignedToUserId}</span>
            </PropertyCell>
            {workItem.assignedAt ? (
              <PropertyCell label={t("Assigned At")}>
                <span className="text-xs font-medium">
                  {formatUnixDateTime(workItem.assignedAt)}
                </span>
              </PropertyCell>
            ) : null}
          </div>
        </div>
      ) : null}

      {workItem.status === "Resolved" || workItem.status === "Dismissed" ? (
        <div className="bg-card rounded-lg border p-3">
          <SectionLabel>{t("Resolution")}</SectionLabel>
          <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
            {workItem.resolutionType ? (
              <PropertyCell label={t("Resolution Type")}>
                <span className="text-xs font-medium">
                  {resolutionTypeChoices.find((c) => c.value === workItem.resolutionType)?.label ??
                    workItem.resolutionType}
                </span>
              </PropertyCell>
            ) : null}
            {workItem.resolvedByUserId ? (
              <PropertyCell label={t("Resolved By")}>
                <span className="text-xs font-medium">{workItem.resolvedByUserId}</span>
              </PropertyCell>
            ) : null}
            {workItem.resolvedAt ? (
              <PropertyCell label={t("Resolved At")}>
                <span className="text-xs font-medium">
                  {formatUnixDateTime(workItem.resolvedAt)}
                </span>
              </PropertyCell>
            ) : null}
          </div>
          {workItem.resolutionNote ? (
            <p className="border-muted-foreground/20 text-muted-foreground mt-2 border-l-2 pl-2.5 text-xs italic">
              {workItem.resolutionNote}
            </p>
          ) : null}
        </div>
      ) : null}

      <div className="bg-card rounded-lg border p-3">
        <SectionLabel>{t("Actions")}</SectionLabel>
        <div className="mt-2">
          {workItem.status === "Open" ? (
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                type="button"
                onClick={() => assignMutation.mutate({ id: workItem.id!, userId: "me" })}
                disabled={assignMutation.isPending}
              >
                <UserPlusIcon className="size-3.5" />
                {t("Assign to Me")}
              </Button>
            </div>
          ) : null}

          {workItem.status === "Assigned" ? (
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                type="button"
                onClick={() => startReviewMutation.mutate(workItem.id!)}
                disabled={startReviewMutation.isPending}
              >
                <PlayIcon className="size-3.5" />
                {t("Start Review")}
              </Button>
            </div>
          ) : null}

          {workItem.status === "InReview" ? (
            <div className="space-y-3">
              {!showResolveForm && !showDismissForm ? (
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    type="button"
                    className="bg-green-600 text-white hover:bg-green-700"
                    onClick={() => {
                      setShowResolveForm(true);
                      setShowDismissForm(false);
                    }}
                  >
                    <ShieldCheckIcon className="size-3.5" />
                    {t("Resolve")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    type="button"
                    onClick={() => {
                      setShowDismissForm(true);
                      setShowResolveForm(false);
                    }}
                  >
                    {t("Dismiss")}
                  </Button>
                </div>
              ) : null}

              {showResolveForm ? (
                <div className="space-y-2">
                  <Select
                    value={resolutionType}
                    onValueChange={(value) => setResolutionType(value as ResolutionType)}
                  >
                    <SelectTrigger className="h-8 text-xs">
                      <SelectValue placeholder={t("Select resolution type")} />
                    </SelectTrigger>
                    <SelectContent>
                      {resolutionTypeChoices.map((choice) => (
                        <SelectItem key={choice.value} value={choice.value}>
                          {t(choice.label)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Textarea
                    value={resolutionNote}
                    onChange={(e) => setResolutionNote(e.target.value)}
                    placeholder={t("Resolution notes...")}
                    className="min-h-[80px] text-xs"
                  />
                  <div className="flex items-center gap-2">
                    <Button
                      size="sm"
                      className="bg-green-600 text-white hover:bg-green-700"
                      type="button"
                      disabled={!resolutionType || resolveMutation.isPending}
                      onClick={() =>
                        resolveMutation.mutate({
                          id: workItem.id!,
                          resolutionType,
                          resolutionNote: resolutionNote.trim(),
                        })
                      }
                    >
                      {t("Confirm Resolution")}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      type="button"
                      onClick={() => {
                        setShowResolveForm(false);
                        setResolutionType("");
                        setResolutionNote("");
                      }}
                    >
                      {t("Cancel")}
                    </Button>
                  </div>
                </div>
              ) : null}

              {showDismissForm ? (
                <div className="space-y-2">
                  <Textarea
                    value={resolutionNote}
                    onChange={(e) => setResolutionNote(e.target.value)}
                    placeholder={t("Dismissal reason...")}
                    className="min-h-[80px] text-xs"
                  />
                  <div className="flex items-center gap-2">
                    <Button
                      size="sm"
                      variant="destructive"
                      type="button"
                      disabled={dismissMutation.isPending}
                      onClick={() =>
                        dismissMutation.mutate({
                          id: workItem.id!,
                          resolutionNote: resolutionNote.trim(),
                        })
                      }
                    >
                      {t("Confirm Dismiss")}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      type="button"
                      onClick={() => {
                        setShowDismissForm(false);
                        setResolutionNote("");
                      }}
                    >
                      {t("Cancel")}
                    </Button>
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}

          {workItem.status === "Resolved" || workItem.status === "Dismissed" ? (
            <p className="text-muted-foreground text-xs">
              {t("This work item has been {0}. No further actions available.", workItem.status.toLowerCase())}
            </p>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-card rounded-lg border px-3 py-2.5">
      <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function SectionLabel({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground text-xs font-medium">{children}</p>;
}

function PropertyCell({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <p className="text-2xs text-muted-foreground">{label}</p>
      {children}
    </div>
  );
}
