import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { PTOStatusBadge, PTOTypeBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRange, getTodayDate, inclusiveDays } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useQuery } from "@tanstack/react-query";
import { CalendarPlusIcon, CalendarRangeIcon, EllipsisIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { canApplyPTOAction } from "./pto/pto-actions";
import { ptoDecision } from "./pto/pto-columns";
import { PTOFormDialog } from "./pto/pto-form-dialog";
import { PTOReasonDialog, type PTOReasonDialogMode } from "./pto/pto-reason-dialog";
import { WorkerPTOBalances } from "./pto/worker-pto-balances";

type PTOSummary = {
  pending: number;
  upcomingApprovedDays: number;
  approvedDaysThisYear: number;
};

export function summarizeWorkerPTO(entries: readonly WorkerPTO[], now: number): PTOSummary {
  const yearStart = new Date(now * 1000);
  yearStart.setMonth(0, 1);
  yearStart.setHours(0, 0, 0, 0);
  const yearStartUnix = Math.floor(yearStart.getTime() / 1000);

  let pending = 0;
  let upcomingApprovedDays = 0;
  let approvedDaysThisYear = 0;
  for (const entry of entries) {
    if (entry.status === "Requested") {
      pending += 1;
      continue;
    }
    if (entry.status !== "Approved") continue;
    const days = inclusiveDays(entry.startDate, entry.endDate);
    if (entry.endDate >= now) {
      upcomingApprovedDays += days;
    }
    if (entry.startDate >= yearStartUnix) {
      approvedDaysThisYear += days;
    }
  }
  return { pending, upcomingApprovedDays, approvedDaysThisYear };
}

type RowActionState =
  | { kind: "edit"; pto: WorkerPTO }
  | { kind: "reason"; pto: WorkerPTO; mode: PTOReasonDialogMode };

export default function WorkerPTOTab({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canCreate } = usePermission(Resource.WorkerPTO, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerPTO, Operation.Update);
  const { allowed: canReject } = usePermission(Resource.WorkerPTO, Operation.Reject);
  const { allowed: canCancel } = usePermission(Resource.WorkerPTO, Operation.Cancel);
  const [requestOpen, setRequestOpen] = useState(false);
  const [rowAction, setRowAction] = useState<RowActionState | null>(null);

  const { data, isLoading } = useQuery(queries.worker.ptoHistory(workerId));
  const entries = useMemo(() => data ?? [], [data]);
  const summary = useMemo(() => summarizeWorkerPTO(entries, getTodayDate()), [entries]);

  if (isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold">{t("Paid Time Off")}</h3>
          <p className="text-muted-foreground text-xs">
            {t("Requests made here or from Dash wait in the approval queue until a dispatcher decides on them.")}
          </p>
        </div>
        {canCreate ? (
          <Button size="sm" onClick={() => setRequestOpen(true)}>
            <CalendarPlusIcon className="size-3.5" />
            {t("Request PTO")}
          </Button>
        ) : null}
      </div>

      <WorkerPTOBalances workerId={workerId} />

      <div className="grid grid-cols-3 gap-2">
        <SummaryTile label={t("Pending requests")} value={summary.pending} />
        <SummaryTile label={t("Upcoming approved days")} value={summary.upcomingApprovedDays} />
        <SummaryTile label={t("Approved days this year")} value={summary.approvedDaysThisYear} />
      </div>

      {entries.length === 0 ? (
        <div className="rounded-lg border border-dashed p-6 text-center">
          <CalendarRangeIcon className="text-muted-foreground mx-auto size-6" />
          <p className="mt-2 text-sm font-medium">{t("No time off on record")}</p>
          <p className="text-muted-foreground mx-auto mt-1 max-w-md text-xs">
            {t("Requests this worker makes in Dash, and any you enter here, will show up in this history.")}
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-xs">
            <thead className="bg-muted/50">
              <tr>
                <th className="px-3 py-2 text-left font-medium">{t("Dates")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("Type")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("Status")}</th>
                <th className="px-3 py-2 text-right font-medium">{t("Days")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("Decision")}</th>
                <th className="w-10 px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => {
                const decision = ptoDecision(entry);
                const showEdit = canUpdate && entry.status === "Requested";
                const showReject = canReject && canApplyPTOAction(entry.status, "Reject");
                const showCancel = canCancel && canApplyPTOAction(entry.status, "Cancel");
                const hasActions = showEdit || showReject || showCancel;
                return (
                  <tr key={entry.id} className="border-t">
                    <td className="px-3 py-2 tabular-nums" title={entry.reason}>
                      {formatRange(entry.startDate, entry.endDate)}
                    </td>
                    <td className="px-3 py-2">
                      <PTOTypeBadge type={entry.type} />
                    </td>
                    <td className="px-3 py-2">
                      <PTOStatusBadge status={entry.status} />
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {inclusiveDays(entry.startDate, entry.endDate)}
                    </td>
                    <td className="text-muted-foreground max-w-[200px] truncate px-3 py-2">
                      {decision
                        ? `${decision.actor}${decision.note ? ` — ${decision.note}` : ""}`
                        : "—"}
                    </td>
                    <td className="px-3 py-2 text-right">
                      {hasActions ? (
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            render={
                              <Button
                                size="sm"
                                variant="ghost"
                                className="size-6"
                                aria-label={t("PTO actions")}
                              >
                                <EllipsisIcon />
                              </Button>
                            }
                          />
                          <DropdownMenuContent side="bottom" align="end">
                            <DropdownMenuGroup>
                              <DropdownMenuLabel>{t("Actions")}</DropdownMenuLabel>
                              <DropdownMenuSeparator />
                              {showEdit ? (
                                <DropdownMenuItem
                                  title={t("Edit")}
                                  description={t("Change the dates, type, or reason")}
                                  onClick={() => setRowAction({ kind: "edit", pto: entry })}
                                />
                              ) : null}
                              {showReject ? (
                                <DropdownMenuItem
                                  title={t("Reject")}
                                  description={t("Reject this PTO request")}
                                  color="danger"
                                  onClick={() =>
                                    setRowAction({ kind: "reason", pto: entry, mode: "reject" })
                                  }
                                />
                              ) : null}
                              {showCancel ? (
                                <DropdownMenuItem
                                  title={t("Cancel")}
                                  description={t("Withdraw this PTO request")}
                                  color="warning"
                                  onClick={() =>
                                    setRowAction({ kind: "reason", pto: entry, mode: "cancel" })
                                  }
                                />
                              ) : null}
                            </DropdownMenuGroup>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      ) : null}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <PTOFormDialog
        open={requestOpen}
        onOpenChange={setRequestOpen}
        defaultWorkerId={workerId}
        lockWorker
      />
      {rowAction?.kind === "edit" ? (
        <PTOFormDialog
          open
          onOpenChange={(open) => {
            if (!open) setRowAction(null);
          }}
          pto={rowAction.pto}
          lockWorker
        />
      ) : null}
      {rowAction?.kind === "reason" ? (
        <PTOReasonDialog
          open
          onOpenChange={(open) => {
            if (!open) setRowAction(null);
          }}
          ptoIds={[rowAction.pto.id ?? ""]}
          mode={rowAction.mode}
        />
      ) : null}
    </div>
  );
}

function SummaryTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="bg-muted/30 rounded-lg border p-3">
      <p className="text-muted-foreground text-[11px] font-medium uppercase">{label}</p>
      <p className="mt-1 text-sm font-semibold tabular-nums">{value}</p>
    </div>
  );
}
