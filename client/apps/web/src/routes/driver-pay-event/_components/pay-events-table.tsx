import { DataTable } from "@/components/data-table/data-table";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { runBulkAction } from "@/lib/bulk-run";
import {
  driverPayEventTableGraphQLConfig,
  holdDriverPayEvent,
  releaseDriverPayEvent,
  type DriverPayEventRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { PauseIcon, PlayIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns, invalidatePayEventQueries } from "./pay-event-columns";
import { PayEventPanel } from "./pay-event-panel";

export default function PayEventsTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [holdRows, setHoldRows] = useState<DriverPayEventRow[]>([]);
  const [holdReason, setHoldReason] = useState("");
  const [holdPending, setHoldPending] = useState(false);

  const handleBulkRelease = useCallback(
    async (rows: DriverPayEventRow[]) => {
      const held = rows.filter((row) => row.onHold);
      if (held.length === 0) {
        toast.info("None of the selected pay events are on hold.");
        return;
      }
      await runBulkAction(held, (row) => releaseDriverPayEvent(row.id), {
        noun: "hold",
        verb: "released",
      });
      invalidatePayEventQueries(queryClient);
    },
    [queryClient],
  );

  const openHoldDialog = useCallback((rows: DriverPayEventRow[]) => {
    const eligible = rows.filter((row) => row.status === "Accrued" && !row.onHold);
    if (eligible.length === 0) {
      toast.info("Only accrued, unheld pay events can be held.");
      return;
    }
    setHoldRows(eligible);
    setHoldReason("");
  }, []);

  const confirmBulkHold = useCallback(async () => {
    setHoldPending(true);
    try {
      await runBulkAction(
        holdRows,
        (row) => holdDriverPayEvent({ payEventId: row.id, reason: holdReason.trim() }),
        { noun: "pay event", verb: "held" },
      );
      invalidatePayEventQueries(queryClient);
      setHoldRows([]);
    } finally {
      setHoldPending(false);
    }
  }, [holdRows, holdReason, queryClient]);

  const dockActions = useMemo<DockAction<DriverPayEventRow>[]>(
    () => [
      {
        id: "hold",
        label: "Hold",
        icon: PauseIcon,
        onClick: openHoldDialog,
      },
      {
        id: "release",
        label: "Release Holds",
        loadingLabel: "Releasing...",
        icon: PlayIcon,
        onClick: handleBulkRelease,
        clearSelectionOnSuccess: true,
      },
    ],
    [openHoldDialog, handleBulkRelease],
  );

  return (
    <>
      <DataTable<DriverPayEventRow>
        name="Pay Event"
        queryKey="driver-pay-event-list"
        graphql={driverPayEventTableGraphQLConfig}
        resource={Resource.DriverSettlement}
        columns={columns}
        dockActions={dockActions}
        enableRowSelection
        TablePanel={PayEventPanel}
      />
      <Dialog open={holdRows.length > 0} onOpenChange={(open) => !open && setHoldRows([])}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Hold {holdRows.length} pay event{holdRows.length === 1 ? "" : "s"}
            </DialogTitle>
            <DialogDescription>
              Held pay skips settlement generation and auto-attach until released. One reason is
              recorded on every selected event.
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={holdReason}
            onChange={(event) => setHoldReason(event.target.value)}
            placeholder="e.g. Awaiting signed BOLs for this batch"
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setHoldRows([])}>
              Cancel
            </Button>
            <Button
              disabled={holdReason.trim() === "" || holdPending}
              onClick={() => void confirmBulkHold()}
            >
              Hold Pay
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
