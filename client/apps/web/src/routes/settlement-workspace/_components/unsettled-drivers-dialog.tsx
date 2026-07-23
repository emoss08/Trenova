import { AmountDisplay } from "@/components/accounting/amount-display";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { getTodayDate } from "@/lib/date";
import {
  fetchUnsettledWorkerSummaries,
  generateDriverSettlement,
  generateSettlementBatch,
  type SettlementWorkspaceSummary,
  type UnsettledWorkerSummary,
} from "@/lib/graphql/driver-settlement";
import { useMutation, useQuery } from "@tanstack/react-query";
import { PauseCircle, Sparkles, Zap } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { InstantPayDialog } from "./instant-pay-dialog";

export function UnsettledDriversDialog({
  open,
  onOpenChange,
  summary,
  onChanged,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  summary: SettlementWorkspaceSummary;
  onChanged: () => void;
}) {
  const [addToBatch, setAddToBatch] = useState(true);
  const [instantPayWorker, setInstantPayWorker] = useState<UnsettledWorkerSummary | null>(null);

  const {
    data: workers,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["unsettled-worker-summaries", summary.periodStart, summary.periodEnd],
    queryFn: () => fetchUnsettledWorkerSummaries(summary.periodStart, summary.periodEnd),
    enabled: open,
  });

  const settleMutation = useMutation({
    mutationFn: (worker: UnsettledWorkerSummary) =>
      generateDriverSettlement({
        workerId: worker.workerId,
        periodStart: summary.periodStart,
        periodEnd: summary.periodEnd,
        payDate: getTodayDate(),
        batchId: addToBatch && summary.openBatchId ? summary.openBatchId : undefined,
      }),
    onSuccess: (settlement, worker) => {
      if (!settlement) {
        toast.info(
          `${worker.workerName} already has a settlement this period — new accruals attach to it automatically`,
        );
        return;
      }
      toast.success(
        `Draft ${settlement.settlementNumber} created for ${worker.workerName} — pay date set to today`,
      );
      void refetch();
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to settle driver"),
  });

  const generateAllMutation = useMutation({
    mutationFn: () => generateSettlementBatch({}),
    onSuccess: (batch) => {
      toast.success(
        `Batch up to date — ${batch.settlementCount} settlement${
          batch.settlementCount === 1 ? "" : "s"
        }${batch.exceptionCount > 0 ? `, ${batch.exceptionCount} need review` : ""}`,
      );
      void refetch();
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to generate settlements"),
  });

  const list = workers ?? [];
  const settleable = list.filter((worker) => !worker.hasSettlement && worker.eventCount > 0);

  return (
    <>
      <InstantPayDialog
        open={instantPayWorker != null}
        onOpenChange={(next) => {
          if (!next) setInstantPayWorker(null);
        }}
        worker={
          instantPayWorker
            ? { workerId: instantPayWorker.workerId, workerName: instantPayWorker.workerName }
            : null
        }
        onPaid={() => {
          void refetch();
          onChanged();
        }}
      />
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Unsettled drivers</DialogTitle>
            <DialogDescription>
              Every driver holding accrued pay this period. Settle a driver individually to pay them
              early, or generate for everyone at once — either way, drivers who already have a
              settlement are skipped and their new accruals attach automatically.
            </DialogDescription>
          </DialogHeader>
          {summary.openBatchId != null && (
            <div className="flex items-center gap-2">
              <Checkbox
                id="unsettled-add-to-batch"
                checked={addToBatch}
                onCheckedChange={(checked) => setAddToBatch(checked === true)}
              />
              <Label htmlFor="unsettled-add-to-batch" className="text-xs font-normal">
                Add off-cycle settlements to the current period batch
              </Label>
            </div>
          )}
          {isLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : list.length === 0 ? (
            <p className="py-4 text-center text-xs text-muted-foreground">
              No drivers have unsettled pay for this period.
            </p>
          ) : (
            <ScrollArea className="max-h-80 min-h-0" viewportClassName="min-h-0" maskHeight={18}>
              <ul className="flex flex-col gap-1.5 pr-2">
                {list.map((worker) => (
                  <li
                    key={worker.workerId}
                    className="flex items-center gap-3 rounded-md border p-2.5"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium">{worker.workerName}</p>
                      <p className="text-[11px] text-muted-foreground">
                        {worker.eventCount} event{worker.eventCount === 1 ? "" : "s"} ·{" "}
                        <AmountDisplay value={worker.grossAmountMinor} currency="USD" />
                        {worker.heldCount > 0 && (
                          <span className="ml-1.5 inline-flex items-center gap-0.5 text-blue-600 dark:text-blue-400">
                            <PauseCircle className="size-3" />
                            {worker.heldCount} held (
                            <AmountDisplay value={worker.heldGrossMinor} currency="USD" />)
                          </span>
                        )}
                      </p>
                    </div>
                    {worker.hasSettlement ? (
                      <span
                        className="text-[10px] text-muted-foreground"
                        title="This driver already has a settlement for the period; accrued pay attaches to their open draft automatically"
                      >
                        Has settlement
                      </span>
                    ) : worker.eventCount === 0 ? (
                      <span
                        className="text-[10px] text-muted-foreground"
                        title="All of this driver's pay is on hold — release it before settling"
                      >
                        All held
                      </span>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <Button
                          size="sm"
                          variant="outline"
                          className="h-7 px-2 text-[11px]"
                          disabled={settleMutation.isPending}
                          onClick={() => settleMutation.mutate(worker)}
                          title="Create a draft settlement for this driver right now — pay date defaults to today for off-cycle pay"
                        >
                          Settle now
                        </Button>
                        <Button
                          size="sm"
                          className="h-7 px-2 text-[11px]"
                          onClick={() => setInstantPayWorker(worker)}
                          title="Pay this driver immediately — approve, post, and mark paid in one pass"
                        >
                          <Zap className="size-3" />
                          Pay now
                        </Button>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </ScrollArea>
          )}
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
              Close
            </Button>
            <Button
              size="sm"
              disabled={generateAllMutation.isPending || settleable.length === 0}
              onClick={() => generateAllMutation.mutate()}
              title="Create draft settlements for every driver listed here in one pass"
            >
              <Sparkles className="size-3.5" />
              Generate for all ({settleable.length})
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
