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
import { runBulkAction } from "@/lib/bulk-run";
import {
  payAdvanceTableGraphQLConfig,
  writeOffPayAdvance,
  type PayAdvanceRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { BanIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./advance-columns";
import { AdvancePanel } from "./advance-panel";

export default function AdvancesTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [writeOffRows, setWriteOffRows] = useState<PayAdvanceRow[]>([]);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);

  const openWriteOffDialog = useCallback((rows: PayAdvanceRow[]) => {
    const eligible = rows.filter(
      (row) => row.status === "Outstanding" || row.status === "PartiallyRecovered",
    );
    if (eligible.length === 0) {
      toast.info(t("Only outstanding or partially recovered advances can be written off."));
      return;
    }
    setWriteOffRows(eligible);
    setReason("");
  }, [t]);

  const confirmWriteOff = useCallback(async () => {
    setPending(true);
    try {
      await runBulkAction(
        writeOffRows,
        (row) => writeOffPayAdvance({ advanceId: row.id, reason: reason.trim() }),
        { noun: "advance", verb: "written off" },
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
        label: "Write Off",
        icon: BanIcon,
        variant: "destructive",
        onClick: openWriteOffDialog,
      },
    ],
    [openWriteOffDialog],
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
              {t("Forgives each advance's remaining balance — nothing more is recovered from settlements. This cannot be undone, and the reason is recorded on every advance.")}
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
              {t("Write Off")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
