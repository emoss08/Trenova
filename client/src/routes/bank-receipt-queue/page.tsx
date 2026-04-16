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
import { Textarea } from "@/components/ui/textarea";
import { resolutionTypeChoices, workItemStatusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  BankReceiptWorkItem,
  ResolutionType,
  WorkItemStatus,
} from "@/types/bank-receipt-work-item";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ClipboardListIcon,
  InboxIcon,
  PlayIcon,
  SearchIcon,
  ShieldCheckIcon,
  UserPlusIcon,
} from "lucide-react";
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
  const [selectedWorkItemId, setSelectedWorkItemId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
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
      toast.success("Work item assigned");
    },
    onError: () => toast.error("Failed to assign work item"),
  });

  const startReviewMutation = useMutation({
    mutationFn: async (id: string) => apiService.bankReceiptWorkItemService.startReview(id),
    onSuccess: () => {
      invalidateAll();
      toast.success("Review started");
    },
    onError: () => toast.error("Failed to start review"),
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
      toast.success("Work item resolved");
    },
    onError: () => toast.error("Failed to resolve work item"),
  });

  const dismissMutation = useMutation({
    mutationFn: async ({ id, resolutionNote }: { id: string; resolutionNote: string }) =>
      apiService.bankReceiptWorkItemService.dismiss(id, { resolutionNote }),
    onSuccess: () => {
      invalidateAll();
      toast.success("Work item dismissed");
    },
    onError: () => toast.error("Failed to dismiss work item"),
  });

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        title: "Bank Receipt Work Queue",
        description: "Review and resolve bank receipt exceptions requiring attention.",
      }}
      toolbar={
        <div className="mx-4 mt-3 grid gap-2.5 md:grid-cols-4">
          <SummaryCard
            label="Open"
            value={String(
              (summaryQuery.data?.activeWorkItemCount ?? 0) -
                (summaryQuery.data?.assignedWorkItemCount ?? 0) -
                (summaryQuery.data?.inReviewWorkItemCount ?? 0),
            )}
          />
          <SummaryCard
            label="Assigned"
            value={String(summaryQuery.data?.assignedWorkItemCount ?? 0)}
          />
          <SummaryCard
            label="In Review"
            value={String(summaryQuery.data?.inReviewWorkItemCount ?? 0)}
          />
          <SummaryCard label="Exceptions" value={String(summaryQuery.data?.exceptionCount ?? 0)} />
        </div>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-1.5 border-b p-2">
            <Input
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Search reference, ID..."
              leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
              className="h-7 text-xs"
            />
            <Select
              value={statusFilter ?? "all"}
              onValueChange={(value) => setStatusFilter(value === "all" ? null : value)}
            >
              <SelectTrigger className="h-7 text-xs">
                <SelectValue placeholder="All statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                {workItemStatusChoices.map((choice) => (
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
                    title="No work items"
                    description="Adjust the filters or wait for new exceptions."
                    icons={[InboxIcon, ClipboardListIcon, ShieldCheckIcon]}
                    className="flex h-full max-w-none flex-col items-center justify-center rounded-none border-none p-6 shadow-none"
                  />
                </div>
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
                        <span className="text-2xs text-muted-foreground">Assigned</span>
                      ) : (
                        <span className="text-2xs text-muted-foreground">Unassigned</span>
                      )}
                    </div>
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
            <div className="flex h-full items-center justify-center">
              <EmptyState
                title="No work item selected"
                description="Select a work item from the list to review and take action."
                icons={[ClipboardListIcon, InboxIcon, ShieldCheckIcon]}
                className="flex h-full flex-col items-center justify-center rounded-none max-w-none border-none p-8 shadow-none"
              />
            </div>
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
          <p className="text-sm text-muted-foreground">Bank Receipt: {workItem.bankReceiptId}</p>
        </div>
        <Badge variant={STATUS_VARIANTS[workItem.status]}>{STATUS_LABELS[workItem.status]}</Badge>
      </div>

      {receipt ? (
        <div className="rounded-lg border bg-card p-3">
          <SectionLabel>Bank Receipt Info</SectionLabel>
          <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
            <PropertyCell label="Receipt Date">
              <span className="text-xs font-medium">
                {new Date(receipt.receiptDate * 1000).toLocaleDateString()}
              </span>
            </PropertyCell>
            <PropertyCell label="Amount">
              <span className="text-xs font-medium tabular-nums">
                {formatCurrency(receipt.amountMinor / 100)}
              </span>
            </PropertyCell>
            <PropertyCell label="Reference">
              <span className="text-xs font-medium">{receipt.referenceNumber}</span>
            </PropertyCell>
            <PropertyCell label="Memo">
              <span className="text-xs font-medium">{receipt.memo || "—"}</span>
            </PropertyCell>
            <PropertyCell label="Status">
              <Badge variant="secondary">{receipt.status}</Badge>
            </PropertyCell>
            {receipt.exceptionReason ? (
              <PropertyCell label="Exception Reason">
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
        <div className="rounded-lg border bg-card p-3">
          <SectionLabel>Assignment</SectionLabel>
          <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
            <PropertyCell label="Assigned To">
              <span className="text-xs font-medium">{workItem.assignedToUserId}</span>
            </PropertyCell>
            {workItem.assignedAt ? (
              <PropertyCell label="Assigned At">
                <span className="text-xs font-medium">
                  {new Date(workItem.assignedAt * 1000).toLocaleString()}
                </span>
              </PropertyCell>
            ) : null}
          </div>
        </div>
      ) : null}

      {workItem.status === "Resolved" || workItem.status === "Dismissed" ? (
        <div className="rounded-lg border bg-card p-3">
          <SectionLabel>Resolution</SectionLabel>
          <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
            {workItem.resolutionType ? (
              <PropertyCell label="Resolution Type">
                <span className="text-xs font-medium">
                  {resolutionTypeChoices.find((c) => c.value === workItem.resolutionType)?.label ??
                    workItem.resolutionType}
                </span>
              </PropertyCell>
            ) : null}
            {workItem.resolvedByUserId ? (
              <PropertyCell label="Resolved By">
                <span className="text-xs font-medium">{workItem.resolvedByUserId}</span>
              </PropertyCell>
            ) : null}
            {workItem.resolvedAt ? (
              <PropertyCell label="Resolved At">
                <span className="text-xs font-medium">
                  {new Date(workItem.resolvedAt * 1000).toLocaleString()}
                </span>
              </PropertyCell>
            ) : null}
          </div>
          {workItem.resolutionNote ? (
            <p className="mt-2 border-l-2 border-muted-foreground/20 pl-2.5 text-xs text-muted-foreground italic">
              {workItem.resolutionNote}
            </p>
          ) : null}
        </div>
      ) : null}

      <div className="rounded-lg border bg-card p-3">
        <SectionLabel>Actions</SectionLabel>
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
                Assign to Me
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
                Start Review
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
                    Resolve
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
                    Dismiss
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
                      <SelectValue placeholder="Select resolution type" />
                    </SelectTrigger>
                    <SelectContent>
                      {resolutionTypeChoices.map((choice) => (
                        <SelectItem key={choice.value} value={choice.value}>
                          {choice.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Textarea
                    value={resolutionNote}
                    onChange={(e) => setResolutionNote(e.target.value)}
                    placeholder="Resolution notes..."
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
                      Confirm Resolution
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
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : null}

              {showDismissForm ? (
                <div className="space-y-2">
                  <Textarea
                    value={resolutionNote}
                    onChange={(e) => setResolutionNote(e.target.value)}
                    placeholder="Dismissal reason..."
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
                      Confirm Dismiss
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
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}

          {workItem.status === "Resolved" || workItem.status === "Dismissed" ? (
            <p className="text-xs text-muted-foreground">
              This work item has been {workItem.status.toLowerCase()}. No further actions available.
            </p>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-card px-3 py-2.5">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
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
