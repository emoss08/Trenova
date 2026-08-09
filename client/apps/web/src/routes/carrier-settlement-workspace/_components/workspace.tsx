import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  fetchCarrierSettlementWorkspaceSummary,
  fetchWorkspaceCarrierSettlements,
  generateCarrierSettlementBatch,
  type CarrierSettlementRow,
} from "@/lib/graphql/carrier-settlement";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { RefreshCcw, Sparkles } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import { CarrierSettlementDetail } from "@/routes/carrier-settlement/_components/settlement-detail";
import { CarrierContextRail } from "./carrier-context-rail";
import { SettlementQueue, type QueueFilter } from "./settlement-queue";
import { WorkspaceSummaryStrip } from "./workspace-summary";

export function invalidateCarrierWorkspace(queryClient: QueryClient) {
  const prefixes = [
    "carrier-settlement-workspace-summary",
    "carrier-settlement-workspace-settlements",
    "carrier-settlement-detail",
    "carrier-settlement-list",
    "carrier-cost-event-list",
    "carrier-pending-cost-events",
    "carrier-recent-settlements",
    "carrier-ledger-entries",
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
  const deepLinkHandled = useRef(false);

  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ["carrier-settlement-workspace-summary"],
    queryFn: () => fetchCarrierSettlementWorkspaceSummary(),
  });

  const periodStart = summary?.periodStart;
  const periodEnd = summary?.periodEnd;
  const { data: settlements, isLoading: settlementsLoading } = useQuery({
    queryKey: ["carrier-settlement-workspace-settlements", periodStart, periodEnd],
    queryFn: () => fetchWorkspaceCarrierSettlements(periodStart as number, periodEnd as number),
    enabled: periodStart != null && periodEnd != null,
  });

  const refresh = useCallback(() => invalidateCarrierWorkspace(queryClient), [queryClient]);

  const filtered = useMemo(() => {
    const list = settlements ?? [];
    if (filter === "all") {
      return list.filter((settlement) => settlement.status !== "Voided");
    }
    return list.filter((settlement) => settlement.status === filter);
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

  const selected: CarrierSettlementRow | null = useMemo(
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
    mutationFn: () => generateCarrierSettlementBatch({}),
    onSuccess: (batch) => {
      toast.success(
        batch.settlementCount > 0
          ? `Batch up to date — ${batch.settlementCount} settlement${
              batch.settlementCount === 1 ? "" : "s"
            }`
          : "Batch created — no carriers had pending cost events",
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
      <WorkspaceSummaryStrip
        summary={summary}
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
              disabled={generateMutation.isPending || summary.pendingEventCount === 0}
              onClick={() => generateMutation.mutate()}
              title={
                summary.pendingEventCount === 0
                  ? "No pending cost events are waiting — there is nothing to generate"
                  : `Build one settlement per carrier from ${summary.pendingEventCount} pending cost events`
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
              <CarrierSettlementDetail
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
          <CarrierContextRail
            carrierId={selected?.carrierId ?? null}
            carrierName={selected?.carrier?.name ?? null}
            selectedSettlement={selected}
          />
        </div>
      ) : (
        <EmptyPeriodState
          pendingEventCount={summary.pendingEventCount}
          pendingCarrierCount={summary.pendingCarrierCount}
          generating={generateMutation.isPending}
          onGenerate={() => generateMutation.mutate()}
        />
      )}
    </div>
  );
}

function EmptyPeriodState({
  pendingEventCount,
  pendingCarrierCount,
  generating,
  onGenerate,
}: {
  pendingEventCount: number;
  pendingCarrierCount: number;
  generating: boolean;
  onGenerate: () => void;
}) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <Sparkles className="size-8 text-muted-foreground" />
      <div>
        <h3 className="text-sm font-semibold">No carrier settlements for this pay period yet</h3>
        <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">
          {pendingEventCount > 0
            ? `${pendingEventCount} cost event${pendingEventCount === 1 ? "" : "s"} across ${pendingCarrierCount} carrier${pendingCarrierCount === 1 ? "" : "s"} are waiting to be settled. Generating builds one draft statement per carrier — linehaul, fuel, and accessorial cost lines are pulled in automatically.`
            : "Cost events accrue automatically as carrier-covered moves complete. Once there is pending cost, generate the period's settlements from here."}
        </p>
      </div>
      {pendingEventCount > 0 && (
        <Button disabled={generating} onClick={onGenerate}>
          <Sparkles className="size-4" />
          Generate Settlements
        </Button>
      )}
    </div>
  );
}
