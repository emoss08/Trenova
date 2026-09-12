import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { AssignPayProfileDialog } from "@/components/pay/assign-pay-profile-dialog";
import { PayeeClassificationBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  endWorkerPayAssignment,
  fetchEffectiveWorkerPayAssignment,
  fetchWorkerEarningsSummary,
  fetchWorkerPayAssignments,
  type EffectiveWorkerPayAssignment,
} from "@/lib/graphql/driver-settlement";
import { formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import type { PayeeClassification } from "@trenova/shared/types/driver-pay";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleDollarSign, Wallet } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

function formatDate(unix?: number | null): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

import { WorkerBenefitsSection } from "./benefits/worker-benefits-section";

export default function WorkerPayTab({ workerId }: { workerId: string }) {
  const t = useT();

  const queryClient = useQueryClient();
  const [assignOpen, setAssignOpen] = useState(false);
  const [endOpen, setEndOpen] = useState(false);

  const { data: assignment, isLoading: assignmentLoading } = useQuery({
    queryKey: ["worker-pay", "effective-assignment", workerId],
    queryFn: ({ signal }) => fetchEffectiveWorkerPayAssignment(workerId, { signal }),
  });
  const { data: history } = useQuery({
    queryKey: ["worker-pay", "assignment-history", workerId],
    queryFn: ({ signal }) => fetchWorkerPayAssignments(workerId, { signal }),
  });
  const { data: earnings } = useQuery({
    queryKey: ["worker-pay", "earnings-summary", workerId],
    queryFn: ({ signal }) => fetchWorkerEarningsSummary(workerId, { signal }),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["worker-pay"] });
  };

  if (assignmentLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold">{t("Pay Profile")}</h3>
          <p className="text-muted-foreground text-xs">
            {t("Pay accrues automatically from delivered shipments using the assignment in effect on the delivery date. Manage shared profiles under Payroll → Pay Profiles.")}
          </p>
        </div>
        <Button size="sm" onClick={() => setAssignOpen(true)}>
          <Wallet className="size-3.5" />
          {assignment ? t("Change Profile") : t("Assign Profile")}
        </Button>
      </div>

      {assignment ? (
        <CurrentAssignmentCard assignment={assignment} onEnd={() => setEndOpen(true)} />
      ) : (
        <div className="rounded-lg border border-dashed p-6 text-center">
          <CircleDollarSign className="text-muted-foreground mx-auto size-6" />
          <p className="mt-2 text-sm font-medium">{t("No pay profile assigned")}</p>
          <p className="text-muted-foreground mx-auto mt-1 max-w-md text-xs">
            {t("This driver will not accrue pay for delivered shipments until a profile is assigned. Assign a shared profile and add driver-specific rate overrides if their rates differ from the template.")}
          </p>
        </div>
      )}

      {earnings && (
        <div className="grid grid-cols-3 gap-2">
          <div className="bg-muted/30 rounded-lg border p-3">
            <p className="text-muted-foreground text-[11px] font-medium uppercase">
              {t("Unsettled Earnings")}
            </p>
            <p className="mt-1 text-sm font-semibold">
              <AmountDisplay value={earnings.accruedGrossMinor} variant="positive" />
            </p>
            <p className="text-muted-foreground text-[11px]">
              {t("{0} pay event {1} awaiting settlement", earnings.accruedEventCount, earnings.accruedEventCount === 1 ? "" : "s")}
            </p>
          </div>
          <div className="bg-muted/30 rounded-lg border p-3">
            <p className="text-muted-foreground text-[11px] font-medium uppercase">
              {t("Outstanding Advances")}
            </p>
            <p className="mt-1 text-sm font-semibold">
              <AmountDisplay
                value={earnings.outstandingAdvances}
                variant={earnings.outstandingAdvances > 0 ? "negative" : "neutral"}
              />
            </p>
            <p className="text-muted-foreground text-[11px]">
              {t("Recovered automatically from the next settlement")}
            </p>
          </div>
          <div className="bg-muted/30 rounded-lg border p-3">
            <p className="text-muted-foreground text-[11px] font-medium uppercase">
              {t("Escrow Balance")}
            </p>
            <p className="mt-1 text-sm font-semibold">
              <AmountDisplay value={earnings.escrowBalanceMinor} />
            </p>
            <p className="text-muted-foreground text-[11px]">
              {t("Ledger under Payroll → Escrow Accounts")}
            </p>
          </div>
        </div>
      )}

      {(history ?? []).length > 0 && (
        <div>
          <h4 className="text-muted-foreground mb-2 text-xs font-semibold tracking-wide uppercase">
            {t("Assignment History")}
          </h4>
          <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-xs">
              <thead className="bg-muted/50 text-left">
                <tr>
                  <th className="px-3 py-2 font-medium">{t("Profile")}</th>
                  <th className="px-3 py-2 font-medium">{t("Effective")}</th>
                  <th className="px-3 py-2 text-right font-medium">{t("Split")}</th>
                  <th className="px-3 py-2 text-right font-medium">{t("Overrides")}</th>
                </tr>
              </thead>
              <tbody>
                {(history ?? []).map((entry) => (
                  <tr key={entry.id} className="border-t">
                    <td className="px-3 py-2 font-medium">{entry.payProfile?.name ?? "—"}</td>
                    <td className="px-3 py-2">
                      {formatDate(entry.effectiveFrom)} –{" "}
                      {entry.effectiveTo ? formatDate(entry.effectiveTo) : "current"}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {Number(entry.splitPercent)}%
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {entry.rateOverrides?.length ?? 0}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <AssignPayProfileDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        workerId={workerId}
        onAssigned={invalidate}
      />
      {assignment && (
        <EndAssignmentDialog
          open={endOpen}
          onOpenChange={setEndOpen}
          assignmentId={assignment.id}
          onEnded={invalidate}
        />
      )}

      <WorkerBenefitsSection workerId={workerId} />
    </div>
  );
}

function CurrentAssignmentCard({
  assignment,
  onEnd,
}: {
  assignment: EffectiveWorkerPayAssignment;
  onEnd: () => void;
}) {
  const t = useT();

  const profile = assignment.payProfile;
  const overrideMap = new Map(
    (assignment.rateOverrides ?? []).map((override) => [override.componentId, override.rate]),
  );
  const activeComponents = (profile?.components ?? []).filter((component) => component.isActive);

  return (
    <div className="rounded-lg border p-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold">{profile?.name ?? t("Pay profile")}</span>
        {profile && (
          <PayeeClassificationBadge
            classification={profile.classification as PayeeClassification}
          />
        )}
        <span className="text-muted-foreground text-xs">
          {t("since {0} {1}", formatDate(assignment.effectiveFrom), Number(assignment.splitPercent) !== 100 &&
            t("· {0}% split", Number(assignment.splitPercent)))}
        </span>
        <Button
          size="sm"
          variant="ghost"
          className="ml-auto text-red-600 dark:text-red-400"
          onClick={onEnd}
        >
          {t("End Assignment")}
        </Button>
      </div>

      {activeComponents.length > 0 && (
        <div className="mt-3 overflow-hidden rounded-md border">
          <table className="w-full text-xs">
            <thead className="bg-muted/50 text-left">
              <tr>
                <th className="px-3 py-1.5 font-medium">{t("Component")}</th>
                <th className="px-3 py-1.5 text-right font-medium">{t("Profile Rate")}</th>
                <th className="px-3 py-1.5 text-right font-medium">{t("This Driver")}</th>
              </tr>
            </thead>
            <tbody>
              {activeComponents.map((component) => {
                const override = overrideMap.get(component.id);
                const suffix = component.method === "PercentOfRevenue" ? "%" : "";
                return (
                  <tr key={component.id} className="border-t">
                    <td className="px-3 py-1.5">
                      {component.description || `${component.kind} (${component.method})`}
                      {(component.bands?.length ?? 0) > 0 && (
                        <span className="text-muted-foreground ml-1">
                          {t("({0} bands)", component.bands?.length)}
                        </span>
                      )}
                    </td>
                    <td className="text-muted-foreground px-3 py-1.5 text-right tabular-nums">
                      {Number(component.rate)}
                      {suffix}
                    </td>
                    <td className="px-3 py-1.5 text-right font-medium tabular-nums">
                      {override != null ? (
                        <span className="text-blue-600 dark:text-blue-400">
                          {t("{0} {1} (override)", Number(override), suffix)}
                        </span>
                      ) : (
                        <>
                          {Number(component.rate)}
                          {suffix}
                        </>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
      {profile != null && profile.guaranteedPeriodMinimumMinor > 0 && (
        <p className="text-muted-foreground mt-2 text-[11px]">
          {t("Guaranteed minimum")} <AmountDisplay value={profile.guaranteedPeriodMinimumMinor} /> {t("per pay period — a top-up line is added automatically when period gross falls below the floor.")}
        </p>
      )}
      <p className="text-muted-foreground mt-2 text-[11px]">
        {t("Need different rates for this driver? Use")}{" "}
        <Link to="/payroll/pay-profiles" className="underline">
          {t("shared profiles")}
        </Link>{" "}
        {t("with per-driver overrides instead of creating one profile per driver.")}
      </p>
    </div>
  );
}

function EndAssignmentDialog({
  open,
  onOpenChange,
  assignmentId,
  onEnded,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  assignmentId: string;
  onEnded: () => void;
}) {
  const t = useT();

  const mutation = useMutation({
    mutationFn: () => endWorkerPayAssignment({ assignmentId, endDate: getTodayDate() }),
    onSuccess: () => {
      toast.success(t("Pay assignment ended"));
      onOpenChange(false);
      onEnded();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to end assignment"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("End pay assignment")}</DialogTitle>
          <DialogDescription>
            {t("The assignment ends today. The driver stops accruing pay for shipments delivered after today until a new profile is assigned; already-accrued pay events are kept.")}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            variant="destructive"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {t("End Assignment")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
