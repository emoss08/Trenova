import { AmountDisplay } from "@/components/accounting/amount-display";
import { DriverPayEventStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import {
  holdDriverPayEvent,
  releaseDriverPayEvent,
  type DriverPayEventRow,
} from "@/lib/graphql/driver-settlement";
import type { DriverPayEventStatus } from "@/types/driver-pay";
import { useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { PauseIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export function invalidatePayEventQueries(queryClient: ReturnType<typeof useQueryClient>) {
  for (const key of [
    "driver-pay-event-list",
    "settlement-workspace-summary",
    "unsettled-worker-summaries",
    "worker-unsettled-events",
  ]) {
    void queryClient.invalidateQueries({ queryKey: [key] });
  }
}

function HoldControls({ row }: { row: DriverPayEventRow }) {
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);

  if (row.status !== "Accrued") return null;

  const release = async () => {
    setPending(true);
    try {
      await releaseDriverPayEvent(row.id);
      toast.success("Hold released — the event will settle normally");
      invalidatePayEventQueries(queryClient);
    } catch (error) {
      toast.error((error as Error).message || "Failed to release hold");
    } finally {
      setPending(false);
    }
  };

  const hold = async () => {
    setPending(true);
    try {
      await holdDriverPayEvent({ payEventId: row.id, reason: reason.trim() });
      toast.success("Pay event held — it will skip settlement generation until released");
      setDialogOpen(false);
      setReason("");
      invalidatePayEventQueries(queryClient);
    } catch (error) {
      toast.error((error as Error).message || "Failed to hold pay event");
    } finally {
      setPending(false);
    }
  };

  if (row.onHold) {
    return (
      <button
        type="button"
        disabled={pending}
        onClick={() => void release()}
        className="inline-flex cursor-pointer rounded-full bg-blue-100 px-1.5 py-px text-[10px] font-medium text-blue-700 hover:bg-blue-200 dark:bg-blue-950 dark:text-blue-300 dark:hover:bg-blue-900"
        title={`On hold: ${row.holdReason}. Click to release so the event settles normally.`}
      >
        Held
      </button>
    );
  }

  return (
    <>
      <button
        type="button"
        disabled={pending}
        onClick={() => setDialogOpen(true)}
        className="inline-flex cursor-pointer items-center rounded-full px-1 py-px text-muted-foreground opacity-0 transition-opacity group-hover/row:opacity-100 hover:text-foreground"
        title="Hold this pay event — it skips settlement generation until released"
        aria-label="Hold pay event"
      >
        <PauseIcon className="size-3" />
      </button>
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Hold pay event</DialogTitle>
            <DialogDescription>
              Held pay skips settlement generation and auto-attach until you release it. The reason
              is shown to anyone reviewing the driver&apos;s pay.
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder="e.g. Awaiting signed BOL from the shipper"
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button disabled={reason.trim() === "" || pending} onClick={() => void hold()}>
              Hold Pay
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function getColumns(): ColumnDef<DriverPayEventRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <span className="group/row flex items-center gap-1">
          <DriverPayEventStatusBadge status={row.original.status as DriverPayEventStatus} />
          <HoldControls row={row.original} />
        </span>
      ),
      size: 140,
      meta: { apiField: "status" },
    },
    {
      id: "worker",
      header: "Driver",
      cell: ({ row }) => (
        <span className="text-xs font-medium">
          {row.original.worker
            ? `${row.original.worker.firstName} ${row.original.worker.lastName}`.trim()
            : "—"}
        </span>
      ),
      size: 180,
    },
    {
      accessorKey: "proNumber",
      header: "Pro #",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.proNumber || "—"}</span>,
      size: 140,
      meta: { apiField: "proNumber" },
    },
    {
      accessorKey: "eventDate",
      header: "Earned",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.eventDate)}</span>,
      size: 110,
      meta: { apiField: "eventDate" },
    },
    {
      accessorKey: "totalMiles",
      header: () => <div className="text-right">Miles</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">
          {Number(row.original.totalMiles).toLocaleString()}
        </div>
      ),
      size: 80,
      meta: { apiField: "totalMiles" },
    },
    {
      id: "components",
      header: "Breakdown",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {(row.original.components ?? [])
            .map((component) => component.description)
            .slice(0, 3)
            .join(" · ") || "—"}
        </span>
      ),
      size: 280,
    },
    {
      accessorKey: "grossAmountMinor",
      header: () => <div className="text-right">Gross Pay</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay
            value={row.original.grossAmountMinor}
            variant="positive"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "grossAmountMinor" },
    },
  ];
}
