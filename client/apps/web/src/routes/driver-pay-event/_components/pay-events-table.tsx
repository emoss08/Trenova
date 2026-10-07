import { useT } from "@trenova/shared/i18n/use-t";
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
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  driverPayEventTableGraphQLConfig,
  holdDriverPayEvent,
  releaseDriverPayEvent,
  type DriverPayEventRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { PauseIcon, PlayIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns, invalidatePayEventQueries } from "./pay-event-columns";
import { PayEventPanel } from "./pay-event-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function PayEventsTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const [holdRows, setHoldRows] = useState<DriverPayEventRow[]>([]);
  const [holdReason, setHoldReason] = useState("");
  const [holdPending, setHoldPending] = useState(false);

  const handleBulkRelease = useCallback(
    async (rows: DriverPayEventRow[]) => {
      const held = rows.filter((row) => row.onHold);
      if (held.length === 0) {
        toast.info(t("None of the selected pay events are on hold."));
        return;
      }
      notifyBulkOutcome(await settleAll(held, (row) => releaseDriverPayEvent(row.id)), {
        succeeded: (count) =>
          translate("{0, plural, one {# hold released} other {# holds released}}", count),
        partial: (succeeded, failed) =>
          translate(
            "{0, plural, one {# hold released} other {# holds released}}, {1} failed",
            succeeded,
            failed,
          ),
        allFailed: (failed) =>
          translate(
            "{0, plural, one {The selected hold failed} other {All # selected holds failed}}",
            failed,
          ),
      });
      invalidatePayEventQueries(queryClient);
    },
    [queryClient, t],
  );

  const openHoldDialog = useCallback(
    (rows: DriverPayEventRow[]) => {
      const eligible = rows.filter((row) => row.status === "Accrued" && !row.onHold);
      if (eligible.length === 0) {
        toast.info(t("Only accrued, unheld pay events can be held."));
        return;
      }
      setHoldRows(eligible);
      setHoldReason("");
    },
    [t],
  );

  const confirmBulkHold = useCallback(async () => {
    setHoldPending(true);
    try {
      notifyBulkOutcome(
        await settleAll(holdRows, (row) =>
          holdDriverPayEvent({ payEventId: row.id, reason: holdReason.trim() }),
        ),
        {
          succeeded: (count) =>
            translate("{0, plural, one {# pay event held} other {# pay events held}}", count),
          partial: (succeeded, failed) =>
            translate(
              "{0, plural, one {# pay event held} other {# pay events held}}, {1} failed",
              succeeded,
              failed,
            ),
          allFailed: (failed) =>
            translate(
              "{0, plural, one {The selected pay event failed} other {All # selected pay events failed}}",
              failed,
            ),
        },
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
        label: t("Hold"),
        icon: PauseIcon,
        onClick: openHoldDialog,
      },
      {
        id: "release",
        label: t("Release holds"),
        loadingLabel: t("Releasing..."),
        icon: PlayIcon,
        onClick: handleBulkRelease,
        clearSelectionOnSuccess: true,
      },
    ],
    [openHoldDialog, handleBulkRelease, t],
  );

  return (
    <>
      <DataTable<DriverPayEventRow>
        name="Pay Event"
        emptyTitle={t("No pay events yet")}
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
              {t("Hold {0, plural, one {# pay event} other {# pay events}}", holdRows.length)}
            </DialogTitle>
            <DialogDescription>
              {t(
                "Held pay skips settlement generation and auto-attach until released. One reason is recorded on every selected event.",
              )}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={holdReason}
            onChange={(event) => setHoldReason(event.target.value)}
            placeholder={t("e.g. Awaiting signed BOLs for this batch")}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setHoldRows([])}>
              {t("Cancel")}
            </Button>
            <Button
              disabled={holdReason.trim() === "" || holdPending}
              onClick={() => void confirmBulkHold()}
            >
              {t("Hold pay")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
