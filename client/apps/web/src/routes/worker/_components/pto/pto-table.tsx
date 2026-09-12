import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { notifyBulkOutcome } from "@/lib/bulk-outcome";
import { bulkWorkerPTOAction } from "@/lib/graphql/worker-mutations";
import { workerTableGraphQLConfigs, type WorkerPTORow } from "@/lib/graphql/worker-table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { PTOBulkAction } from "@trenova/shared/types/worker";
import { BanIcon, CircleCheckIcon, CircleXIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import {
  PTO_ACTION_LABELS,
  bulkPayloadToOutcome,
  canApplyPTOAction,
  ptoIds,
  splitPTOByAction,
} from "./pto-actions";
import { getColumns } from "./pto-columns";
import { PTOPanel } from "./pto-panel";
import { PTOReasonDialog, type PTOReasonDialogMode } from "./pto-reason-dialog";
import { usePTOInvalidation } from "./use-pto-invalidation";

type PendingApproval = {
  ids: string[];
  skipped: number;
};

type PendingReason = {
  mode: PTOReasonDialogMode;
  ids: string[];
  skipped: number;
};

const REASON_MODE: Record<Exclude<PTOBulkAction, "Approve">, PTOReasonDialogMode> = {
  Reject: "reject",
  Cancel: "cancel",
};

export default function PTODataTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);
  const invalidate = usePTOInvalidation();
  const { allowed: canApprove } = usePermission(Resource.WorkerPTO, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.WorkerPTO, Operation.Reject);
  const { allowed: canCancel } = usePermission(Resource.WorkerPTO, Operation.Cancel);

  const [pendingApproval, setPendingApproval] = useState<PendingApproval | null>(null);
  const [approving, setApproving] = useState(false);
  const [pendingReason, setPendingReason] = useState<PendingReason | null>(null);

  const openAction = useCallback((rows: readonly WorkerPTORow[], action: PTOBulkAction) => {
    const { eligible, ineligible } = splitPTOByAction(rows, action);
    if (eligible.length === 0) {
      toast.info(PTO_ACTION_LABELS[action].noneEligible);
      return;
    }

    const ids = ptoIds(eligible);
    if (action === "Approve") {
      setPendingApproval({ ids, skipped: ineligible.length });
      return;
    }
    setPendingReason({ mode: REASON_MODE[action], ids, skipped: ineligible.length });
  }, []);

  const confirmApproval = useCallback(async () => {
    if (!pendingApproval) return;
    setApproving(true);
    try {
      const payload = await bulkWorkerPTOAction({
        ptoIds: pendingApproval.ids,
        action: "Approve",
      });
      notifyBulkOutcome(bulkPayloadToOutcome(payload), {
        entity: "PTO request",
        verbPast: PTO_ACTION_LABELS.Approve.verbPast,
        skipped: pendingApproval.skipped,
      });
      await invalidate();
      setPendingApproval(null);
    } catch (error) {
      toast.error(t("Failed to approve PTO"), {
        description: error instanceof Error ? error.message : "Please try again.",
      });
    } finally {
      setApproving(false);
    }
  }, [invalidate, pendingApproval, t]);

  const dockActions = useMemo<DockAction<WorkerPTORow>[]>(() => {
    const actions: DockAction<WorkerPTORow>[] = [];
    if (canApprove) {
      actions.push({
        id: "approve",
        label: PTO_ACTION_LABELS.Approve.label,
        icon: CircleCheckIcon,
        onClick: (rows) => openAction(rows, "Approve"),
      });
    }
    if (canReject) {
      actions.push({
        id: "reject",
        label: PTO_ACTION_LABELS.Reject.label,
        icon: CircleXIcon,
        onClick: (rows) => openAction(rows, "Reject"),
      });
    }
    if (canCancel) {
      actions.push({
        id: "cancel",
        label: PTO_ACTION_LABELS.Cancel.label,
        icon: BanIcon,
        variant: "destructive",
        onClick: (rows) => openAction(rows, "Cancel"),
      });
    }
    return actions;
  }, [canApprove, canCancel, canReject, openAction]);

  const contextMenuActions = useMemo<RowAction<WorkerPTORow>[]>(() => {
    const actions: RowAction<WorkerPTORow>[] = [];
    if (canApprove) {
      actions.push({
        id: "approve",
        label: PTO_ACTION_LABELS.Approve.label,
        icon: CircleCheckIcon,
        group: "decision",
        hidden: (row) => !canApplyPTOAction(row.original.status, "Approve"),
        onClick: (row) => openAction([row.original], "Approve"),
      });
    }
    if (canReject) {
      actions.push({
        id: "reject",
        label: PTO_ACTION_LABELS.Reject.label,
        icon: CircleXIcon,
        group: "decision",
        hidden: (row) => !canApplyPTOAction(row.original.status, "Reject"),
        onClick: (row) => openAction([row.original], "Reject"),
      });
    }
    if (canCancel) {
      actions.push({
        id: "cancel",
        label: PTO_ACTION_LABELS.Cancel.label,
        icon: BanIcon,
        variant: "destructive",
        hidden: (row) => !canApplyPTOAction(row.original.status, "Cancel"),
        onClick: (row) => openAction([row.original], "Cancel"),
      });
    }
    return actions;
  }, [canApprove, canCancel, canReject, openAction]);

  const approvalCount = pendingApproval?.ids.length ?? 0;

  return (
    <>
      <DataTable<WorkerPTORow>
        queryKey="worker-pto-list"
        name="Worker PTO"
        resource={Resource.WorkerPTO}
        columns={columns}
        graphql={workerTableGraphQLConfigs.pto}
        enableRowSelection={dockActions.length > 0}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        TablePanel={PTOPanel}
        enableReadOnlyPanel
      />
      <AlertDialog
        open={pendingApproval !== null}
        onOpenChange={(open) => {
          if (!open && !approving) {
            setPendingApproval(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <CircleCheckIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>
              {t("Approve {0, plural, one {# PTO request} other {# PTO requests}}", approvalCount)}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Workers are notified in Dash and by SMS once their time off is approved. {0}",
                pendingApproval && pendingApproval.skipped > 0
                  ? ` ${t("{0} selected request{1} not eligible and will be skipped.", pendingApproval.skipped, pendingApproval.skipped === 1 ? " is" : t("s are"))}`
                  : "",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={approving}>{t("Back")}</AlertDialogCancel>
            <AlertDialogAction
              disabled={approving}
              onClick={(event) => {
                event.preventDefault();
                void confirmApproval();
              }}
            >
              {approving ? PTO_ACTION_LABELS.Approve.loadingLabel : t("Approve")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      {pendingReason ? (
        <PTOReasonDialog
          open
          onOpenChange={(open) => {
            if (!open) setPendingReason(null);
          }}
          ptoIds={pendingReason.ids}
          mode={pendingReason.mode}
          skipped={pendingReason.skipped}
        />
      ) : null}
    </>
  );
}
