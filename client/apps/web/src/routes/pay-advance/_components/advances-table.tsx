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
  payAdvanceTableGraphQLConfig,
  writeOffPayAdvance,
  type PayAdvanceRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { SlashCircle01Icon } from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./advance-columns";
import { AdvancePanel } from "./advance-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function AdvancesTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const [writeOffRows, setWriteOffRows] = useState<PayAdvanceRow[]>([]);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);

  const openWriteOffDialog = useCallback(
    (rows: PayAdvanceRow[]) => {
      const eligible = rows.filter(
        (row) => row.status === "Outstanding" || row.status === "PartiallyRecovered",
      );
      if (eligible.length === 0) {
        toast.info(t("Only outstanding or partially recovered advances can be written off."));
        return;
      }
      setWriteOffRows(eligible);
      setReason("");
    },
    [t],
  );

  const confirmWriteOff = useCallback(async () => {
    setPending(true);
    try {
      notifyBulkOutcome(
        await settleAll(writeOffRows, (row) =>
          writeOffPayAdvance({ advanceId: row.id, reason: reason.trim() }),
        ),
        {
          succeeded: (count) =>
            translate(
              "{0, plural, one {# advance written off} other {# advances written off}}",
              count,
            ),
          partial: (succeeded, failed) =>
            translate(
              "{0, plural, one {# advance written off} other {# advances written off}}, {1} failed",
              succeeded,
              failed,
            ),
          allFailed: (failed) =>
            translate(
              "{0, plural, one {The selected advance failed} other {All # selected advances failed}}",
              failed,
            ),
        },
      );
      await queryClient.invalidateQueries({ queryKey: ["pay-advance-list"] });
      await queryClient.invalidateQueries({ queryKey: ["worker-pay-advances"] });
      setWriteOffRows([]);
    } finally {
      setPending(false);
    }
  }, [writeOffRows, reason, queryClient]);

  const dockActions = useMemo<DockAction<PayAdvanceRow>[]>(
    () => [
      {
        id: "write-off",
        label: t("Write off"),
        icon: SlashCircle01Icon,
        variant: "destructive",
        onClick: openWriteOffDialog,
      },
    ],
    [openWriteOffDialog, t],
  );

  return (
    <>
      <DataTable<PayAdvanceRow>
        name="Pay Advance"
        queryKey="pay-advance-list"
        graphql={payAdvanceTableGraphQLConfig}
        resource={Resource.PayAdvance}
        columns={columns}
        dockActions={dockActions}
        enableRowSelection
        TablePanel={AdvancePanel}
      />
      <Dialog open={writeOffRows.length > 0} onOpenChange={(open) => !open && setWriteOffRows([])}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t("Write off {0, plural, one {# advance} other {# advances}}", writeOffRows.length)}
            </DialogTitle>
            <DialogDescription>
              {t(
                "Forgives each advance's remaining balance — nothing more is recovered from settlements. This cannot be undone, and the reason is recorded on every advance.",
              )}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder={t("e.g. Driver terminated — balance uncollectible")}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setWriteOffRows([])}>
              {t("Cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={reason.trim() === "" || pending}
              onClick={() => void confirmWriteOff()}
            >
              {t("Write off")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
