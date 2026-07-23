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
import { getTodayDate } from "@trenova/shared/lib/date";
import type { PayeeClassification } from "@trenova/shared/types/driver-pay";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleDollarSign, Wallet } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function WorkerPayTab({ workerId }: { workerId: string }) {
  const queryClient = useQueryClient();
  const [assignOpen, setAssignOpen] = useState(false);
  const [endOpen, setEndOpen] = useState(false);

  const { data: assignment, isLoading: assignmentLoading } = useQuery({
    queryKey: ["worker-pay", "effective-assignment", workerId],
    queryFn: () => fetchEffectiveWorkerPayAssignment(workerId),
  });
  const { data: history } = useQuery({
    queryKey: ["worker-pay", "assignment-history", workerId],
    queryFn: () => fetchWorkerPayAssignments(workerId),
  });
  const { data: earnings } = useQuery({
    queryKey: ["worker-pay", "earnings-summary", workerId],
    queryFn: () => fetchWorkerEarningsSummary(workerId),
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
          <h3 className="text-sm font-semibold">Pay Profile</h3>
          <p className="text-xs text-muted-foreground">
            Pay accrues automatically from delivered shipments using the assignment in effect on the
            delivery date. Manage shared profiles under Payroll &rarr; Pay Profiles.
          </p>
        </div>
        <Button size="sm" onClick={() => setAssignOpen(true)}>
          <Wallet className="size-3.5" />
          {assignment ? "Change Profile" : "Assign Profile"}
        </Button>
      </div>

      {assignment ? (
        <CurrentAssignmentCard assignment={assignment} onEnd={() => setEndOpen(true)} />
      ) : (
        <div className="rounded-lg border border-dashed p-6 text-center">
          <CircleDollarSign className="mx-auto size-6 text-muted-foreground" />
          <p className="mt-2 text-sm font-medium">No pay profile assigned</p>
          <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">
            This driver will not accrue pay for delivered shipments until a profile is assigned.
            Assign a shared profile and add driver-specific rate overrides if their rates differ
            from the template.
          </p>
        </div>
      )}

      {earnings && (
        <div className="grid grid-cols-3 gap-2">
          <div className="rounded-lg border bg-muted/30 p-3">
            <p className="text-[11px] font-medium text-muted-foreground uppercase">
              Unsettled Earnings
            </p>
            <p className="mt-1 text-sm font-semibold">
              <AmountDisplay value={earnings.accruedGrossMinor} variant="positive" />
            </p>
            <p className="text-[11px] text-muted-foreground">
              {earnings.accruedEventCount} pay event
              {earnings.accruedEventCount === 1 ? "" : "s"} awaiting settlement
            </p>
          </div>
          <div className="rounded-lg border bg-muted/30 p-3">
            <p className="text-[11px] font-medium text-muted-foreground uppercase">
              Outstanding Advances
            </p>
            <p className="mt-1 text-sm font-semibold">
              <AmountDisplay
                value={earnings.outstandingAdvances}
                variant={earnings.outstandingAdvances > 0 ? "negative" : "neutral"}
              />
            </p>
            <p className="text-[11px] text-muted-foreground">
              Recovered automatically from the next settlement
            </p>
          </div>
          <div className="rounded-lg border bg-muted/30 p-3">
            <p className="text-[11px] font-medium text-muted-foreground uppercase">
              Escrow Balance
            </p>
            <p className="mt-1 text-sm font-semibold">
              <AmountDisplay value={earnings.escrowBalanceMinor} />
            </p>
            <p className="text-[11px] text-muted-foreground">
              Ledger under Payroll &rarr; Escrow Accounts
            </p>
          </div>
        </div>
      )}

      {(history ?? []).length > 0 && (
        <div>
          <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
            Assignment History
          </h4>
          <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-xs">
              <thead className="bg-muted/50 text-left">
                <tr>
                  <th className="px-3 py-2 font-medium">Profile</th>
                  <th className="px-3 py-2 font-medium">Effective</th>
                  <th className="px-3 py-2 text-right font-medium">Split</th>
                  <th className="px-3 py-2 text-right font-medium">Overrides</th>
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
  const profile = assignment.payProfile;
  const overrideMap = new Map(
    (assignment.rateOverrides ?? []).map((override) => [override.componentId, override.rate]),
  );
  const activeComponents = (profile?.components ?? []).filter((component) => component.isActive);

  return (
    <div className="rounded-lg border p-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold">{profile?.name ?? "Pay profile"}</span>
        {profile && (
          <PayeeClassificationBadge
            classification={profile.classification as PayeeClassification}
          />
        )}
        <span className="text-xs text-muted-foreground">
          since {formatDate(assignment.effectiveFrom)}
          {Number(assignment.splitPercent) !== 100 &&
            ` · ${Number(assignment.splitPercent)}% split`}
        </span>
        <Button
          size="sm"
          variant="ghost"
          className="ml-auto text-red-600 dark:text-red-400"
          onClick={onEnd}
        >
          End Assignment
        </Button>
      </div>

      {activeComponents.length > 0 && (
        <div className="mt-3 overflow-hidden rounded-md border">
          <table className="w-full text-xs">
            <thead className="bg-muted/50 text-left">
              <tr>
                <th className="px-3 py-1.5 font-medium">Component</th>
                <th className="px-3 py-1.5 text-right font-medium">Profile Rate</th>
                <th className="px-3 py-1.5 text-right font-medium">This Driver</th>
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
                        <span className="ml-1 text-muted-foreground">
                          ({component.bands?.length} bands)
                        </span>
                      )}
                    </td>
                    <td className="px-3 py-1.5 text-right text-muted-foreground tabular-nums">
                      {Number(component.rate)}
                      {suffix}
                    </td>
                    <td className="px-3 py-1.5 text-right font-medium tabular-nums">
                      {override != null ? (
                        <span className="text-blue-600 dark:text-blue-400">
                          {Number(override)}
                          {suffix} (override)
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
        <p className="mt-2 text-[11px] text-muted-foreground">
          Guaranteed minimum <AmountDisplay value={profile.guaranteedPeriodMinimumMinor} /> per pay
          period — a top-up line is added automatically when period gross falls below the floor.
        </p>
      )}
      <p className="mt-2 text-[11px] text-muted-foreground">
        Need different rates for this driver? Use{" "}
        <Link to="/payroll/pay-profiles" className="underline">
          shared profiles
        </Link>{" "}
        with per-driver overrides instead of creating one profile per driver.
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
  const mutation = useMutation({
    mutationFn: () => endWorkerPayAssignment({ assignmentId, endDate: getTodayDate() }),
    onSuccess: () => {
      toast.success("Pay assignment ended");
      onOpenChange(false);
      onEnded();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to end assignment"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>End pay assignment</DialogTitle>
          <DialogDescription>
            The assignment ends today. The driver stops accruing pay for shipments delivered after
            today until a new profile is assigned; already-accrued pay events are kept.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            End Assignment
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
