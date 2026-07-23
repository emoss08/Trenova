import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchSettlementWorkspaceSummary,
  fetchWorkspaceSettlements,
  generateSettlementBatch,
  type DriverSettlementRow,
} from "@/lib/graphql/driver-settlement";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { RefreshCcw, Sparkles, Zap } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import { SettlementDetail } from "@/routes/driver-settlement/_components/settlement-detail";
import { DriverContextRail } from "./driver-context-rail";
import { SettlementQueue, type QueueFilter } from "./settlement-queue";
import { InstantPayDialog } from "./instant-pay-dialog";
import { UnsettledDriversDialog } from "./unsettled-drivers-dialog";
import { WorkspaceSummaryStrip } from "./workspace-summary";

export function invalidateWorkspace(queryClient: QueryClient) {
  const prefixes = [
    "settlement-workspace-summary",
    "settlement-workspace-settlements",
    "driver-settlement-detail",
    "driver-settlement-list",
    "driver-pay-event-list",
    "worker-unsettled-events",
    "worker-earnings-summary",
    "worker-recurring-deductions",
    "worker-recurring-earnings",
    "worker-pay-advances",
    "unsettled-worker-summaries",
  ];
  for (const prefix of prefixes) {
    void queryClient.invalidateQueries({ queryKey: [prefix] });
  }
}

export default function Workspace() {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [filter, setFilter] = useState<QueueFilter>("all");
  const [selectedId, setSelectedId] = useState<string | null>(
    () => searchParams.get("settlement") ?? null,
  );
  const [checkedIds, setCheckedIds] = useState<ReadonlySet<string>>(new Set());
  const [showUnsettled, setShowUnsettled] = useState(false);
  const [showInstantPay, setShowInstantPay] = useState(false);
  const deepLinkHandled = useRef(false);

  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ["settlement-workspace-summary"],
    queryFn: () => fetchSettlementWorkspaceSummary(),
  });

  const periodStart = summary?.periodStart;
  const periodEnd = summary?.periodEnd;
  const { data: settlements, isLoading: settlementsLoading } = useQuery({
    queryKey: ["settlement-workspace-settlements", periodStart, periodEnd],
    queryFn: () => fetchWorkspaceSettlements(periodStart as number, periodEnd as number),
    enabled: periodStart != null && periodEnd != null,
  });

  const refresh = useCallback(() => invalidateWorkspace(queryClient), [queryClient]);

  const filtered = useMemo(() => {
    const list = settlements ?? [];
    switch (filter) {
      case "all":
        return list.filter((settlement) => settlement.status !== "Voided");
      case "attention":
        return list.filter(
          (settlement) =>
            settlement.hasExceptions &&
            settlement.status !== "Voided" &&
            settlement.status !== "Paid",
        );
      default:
        return list.filter((settlement) => settlement.status === filter);
    }
  }, [settlements, filter]);

  useEffect(() => {
    if (settlements == null || deepLinkHandled.current) return;
    const target = searchParams.get("settlement");
    if (!target) {
      deepLinkHandled.current = true;
      return;
    }
    deepLinkHandled.current = true;
    if (!settlements.some((settlement) => settlement.id === target)) {
      toast.info(
        "That settlement isn't in the current pay period — look it up in Settlement History.",
      );
    }
    setSearchParams({}, { replace: true });
  }, [settlements, searchParams, setSearchParams]);

  useEffect(() => {
    if (settlements == null) return;
    if (filtered.length === 0) {
      setSelectedId(null);
      return;
    }
    if (!selectedId || !filtered.some((settlement) => settlement.id === selectedId)) {
      setSelectedId(filtered[0].id);
    }
  }, [filtered, selectedId, settlements]);

  const selected: DriverSettlementRow | null = useMemo(
    () => (settlements ?? []).find((settlement) => settlement.id === selectedId) ?? null,
    [settlements, selectedId],
  );

  const selectNext = useCallback(() => {
    const index = filtered.findIndex((settlement) => settlement.id === selectedId);
    if (index >= 0 && index < filtered.length - 1) {
      setSelectedId(filtered[index + 1].id);
    }
  }, [filtered, selectedId]);

  const generateMutation = useMutation({
    mutationFn: () => generateSettlementBatch({}),
    onSuccess: (batch) => {
      toast.success(
        batch.settlementCount > 0
          ? `Batch up to date — ${batch.settlementCount} settlement${
              batch.settlementCount === 1 ? "" : "s"
            }${batch.exceptionCount > 0 ? `, ${batch.exceptionCount} need review` : ""}`
          : "Batch created — no drivers had unsettled pay events",
      );
      refresh();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to generate settlements"),
  });

  if (summaryLoading || !summary) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  const hasSettlements = (settlements ?? []).length > 0;

  return (
    <div className="flex h-[calc(100vh-9.5rem)] min-h-135 flex-col gap-3">
      <UnsettledDriversDialog
        open={showUnsettled}
        onOpenChange={setShowUnsettled}
        summary={summary}
        onChanged={refresh}
      />
      <InstantPayDialog open={showInstantPay} onOpenChange={setShowInstantPay} onPaid={refresh} />
      <WorkspaceSummaryStrip
        summary={summary}
        onFilterAttention={() => setFilter("attention")}
        onShowUnsettled={() => setShowUnsettled(true)}
        actions={
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={refresh}
              aria-label="Refresh workspace data"
            >
              <RefreshCcw className="size-3.5" />
              Refresh
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setShowInstantPay(true)}
              title="Pay a driver immediately — builds, approves, posts, and pays an off-cycle settlement in one pass"
            >
              <Zap className="size-3.5" />
              Pay Now
            </Button>
            <Button
              size="sm"
              disabled={generateMutation.isPending || summary.unsettledEventCount === 0}
              onClick={() => generateMutation.mutate()}
              title={
                summary.unsettledEventCount === 0
                  ? "No unsettled pay events are waiting — there is nothing to generate"
                  : `Build one settlement per driver from ${summary.unsettledEventCount} unsettled pay events`
              }
            >
              <Sparkles className="size-3.5" />
              Generate Settlements
            </Button>
          </div>
        }
      />
      {hasSettlements || settlementsLoading ? (
        <div className="grid min-h-0 flex-1 grid-cols-1 gap-3 lg:grid-cols-[300px_minmax(0,1fr)_300px]">
          <SettlementQueue
            settlements={filtered}
            allSettlements={settlements ?? []}
            loading={settlementsLoading}
            filter={filter}
            onFilterChange={setFilter}
            selectedId={selectedId}
            onSelect={setSelectedId}
            checkedIds={checkedIds}
            onCheckedChange={setCheckedIds}
            onActionComplete={refresh}
          />
          <div className="min-h-0 overflow-hidden rounded-lg border bg-card">
            {selected ? (
              <SettlementDetail
                key={selected.id}
                settlementId={selected.id}
                onClose={selectNext}
                scrollMaskVariant="card"
              />
            ) : (
              <div className="flex h-full items-center justify-center p-8 text-center text-sm text-muted-foreground">
                Select a settlement from the queue to work it here.
              </div>
            )}
          </div>
          <DriverContextRail
            workerId={selected?.workerId ?? null}
            workerName={
              selected?.worker
                ? `${selected.worker.firstName} ${selected.worker.lastName}`.trim()
                : null
            }
            selectedSettlement={selected}
            onChanged={refresh}
          />
        </div>
      ) : (
        <EmptyPeriodState
          unsettledEventCount={summary.unsettledEventCount}
          unsettledWorkerCount={summary.unsettledWorkerCount}
          generating={generateMutation.isPending}
          onGenerate={() => generateMutation.mutate()}
        />
      )}
    </div>
  );
}

function EmptyPeriodState({
  unsettledEventCount,
  unsettledWorkerCount,
  generating,
  onGenerate,
}: {
  unsettledEventCount: number;
  unsettledWorkerCount: number;
  generating: boolean;
  onGenerate: () => void;
}) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <Sparkles className="size-8 text-muted-foreground" />
      <div>
        <h3 className="text-sm font-semibold">No settlements for this pay period yet</h3>
        <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">
          {unsettledEventCount > 0
            ? `${unsettledEventCount} pay event${unsettledEventCount === 1 ? "" : "s"} across ${unsettledWorkerCount} driver${unsettledWorkerCount === 1 ? "" : "s"} are waiting to be settled. Generating builds one draft settlement per driver — earnings, deductions, advance recoveries, and escrow are pulled in automatically.`
            : "Pay events accrue automatically as drivers complete moves. Once there is unsettled pay, generate the period's settlements from here."}
        </p>
      </div>
      {unsettledEventCount > 0 && (
        <Button disabled={generating} onClick={onGenerate}>
          <Sparkles className="size-4" />
          Generate Settlements
        </Button>
      )}
    </div>
  );
}
