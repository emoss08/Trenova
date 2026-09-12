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
import { runBulkAction } from "@/lib/bulk-run";
import {
  closeEscrowAccount,
  escrowAccountTableGraphQLConfig,
  type EscrowAccountRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./escrow-columns";
import { EscrowPanel } from "./escrow-panel";

export default function EscrowTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [closeRows, setCloseRows] = useState<EscrowAccountRow[]>([]);
  const [pending, setPending] = useState(false);

  const openCloseDialog = useCallback((rows: EscrowAccountRow[]) => {
    const eligible = rows.filter((row) => row.status === "Active");
    if (eligible.length === 0) {
      toast.info(t("Every selected escrow account is already closed."));
      return;
    }
    setCloseRows(eligible);
  }, [t]);

  const confirmClose = useCallback(async () => {
    setPending(true);
    try {
      await runBulkAction(closeRows, (row) => closeEscrowAccount(row.id), {
        noun: "escrow account",
        verb: "closed",
      });
      await queryClient.invalidateQueries({ queryKey: ["escrow-account-list"] });
      setCloseRows([]);
    } finally {
      setPending(false);
    }
  }, [closeRows, queryClient]);

  const dockActions = useMemo<DockAction<EscrowAccountRow>[]>(
    () => [
      {
        id: "close",
        label: "Close Accounts",
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: openCloseDialog,
      },
    ],
    [openCloseDialog],
  );

  const withBalance = closeRows.filter((row) => row.balanceMinor !== 0).length;

  return (
    <>
      <DataTable<EscrowAccountRow>
        name="Escrow Account"
        queryKey="escrow-account-list"
        graphql={escrowAccountTableGraphQLConfig}
        resource={Resource.EscrowAccount}
        columns={columns}
        dockActions={dockActions}
        enableRowSelection
        TablePanel={EscrowPanel}
      />
      <Dialog open={closeRows.length > 0} onOpenChange={(open) => !open && setCloseRows([])}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t("Close {0, plural, one {# escrow account} other {# escrow accounts}}", closeRows.length)}
            </DialogTitle>
            <DialogDescription>
              {t("Closed accounts stop accepting contributions and accruing interest. Refund or apply each balance first — accounts holding funds cannot be closed.")}
            </DialogDescription>
          </DialogHeader>
          {withBalance > 0 && (
            <p className="text-xs text-amber-600 dark:text-amber-400">
              {t("{0} selected account{1} a balance and will fail to close until the funds are refunded or applied.", withBalance, withBalance === 1 ? ` ${t("still holds")}` : t("s still hold"))}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setCloseRows([])}>
              {t("Cancel")}
            </Button>
            <Button variant="destructive" disabled={pending} onClick={() => void confirmClose()}>
              {t("Close Accounts")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
