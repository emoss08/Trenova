import { useT } from "@trenova/shared/i18n/use-t";
import { formatPtoDays } from "@trenova/shared/lib/pto";
import { InfoPopover } from "@/components/info-popover";
import { usePermission } from "@/hooks/use-permission";
import { ptoTypeChoices } from "@/lib/choices";
import {
  fetchWorkerPtoBalances,
  fetchWorkerPtoLedger,
  fetchWorkerPtoPolicyAssignments,
  PTO_BALANCE_SUMMARY_KEY,
  runPtoAccrual,
  WORKER_PTO_ASSIGNMENTS_KEY,
  WORKER_PTO_BALANCES_KEY,
  WORKER_PTO_LEDGER_KEY,
  type PTOPolicyAssignment,
  type WorkerPTOBalanceView,
  type WorkerPTOLedgerEntry,
} from "@/lib/graphql/pto-policy";
import { Badge } from "@trenova/shared/components/ui/badge";
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
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PTO_LEDGER_ENTRY_LABELS } from "@trenova/shared/types/pto-policy";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarSyncIcon, EllipsisIcon, ScaleIcon, ShieldCheckIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { AdjustBalanceDialog } from "./adjust-balance-dialog";
import { AssignPolicyDialog } from "./assign-policy-dialog";

const LEDGER_PAGE_SIZE = 25;

function typeLabel(ptoType: string): string {
  return ptoTypeChoices.find((choice) => choice.value === ptoType)?.label ?? ptoType;
}

export function useWorkerPtoInvalidation(workerId: string) {
  const queryClient = useQueryClient();
  return useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [WORKER_PTO_BALANCES_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: [WORKER_PTO_LEDGER_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: [WORKER_PTO_ASSIGNMENTS_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: [PTO_BALANCE_SUMMARY_KEY] }),
    ]);
  }, [queryClient, workerId]);
}

export function WorkerPTOBalances({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canAssign } = usePermission(Resource.PTOPolicy, Operation.Assign);
  const { allowed: canManage } = usePermission(Resource.WorkerPTO, Operation.Manage);
  const invalidate = useWorkerPtoInvalidation(workerId);
  const [assignOpen, setAssignOpen] = useState(false);
  const [adjustOpen, setAdjustOpen] = useState(false);

  const balancesQuery = useQuery({
    queryKey: [WORKER_PTO_BALANCES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPtoBalances(workerId, { signal }),
  });
  const assignmentsQuery = useQuery({
    queryKey: [WORKER_PTO_ASSIGNMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPtoPolicyAssignments(workerId, { signal }),
  });

  const balances = useMemo(() => balancesQuery.data ?? [], [balancesQuery.data]);
  const assignments = useMemo(() => assignmentsQuery.data ?? [], [assignmentsQuery.data]);
  const current = useMemo(
    () => assignments.find((assignment) => assignment.effectiveTo == null) ?? null,
    [assignments],
  );

  const accrual = useMutation({
    mutationFn: () => runPtoAccrual({ workerId }),
    onSuccess: (result) => {
      toast.success(t("Accrual run complete"), {
        description: `${result.entriesPosted} posted, ${result.entriesCapped} capped, ${result.entriesSkipped} already posted.`,
      });
      void invalidate();
    },
    onError: (error: Error) => {
      toast.error(t("Accrual run failed"), { description: error.message });
    },
  });

  if (balancesQuery.isLoading || assignmentsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-2">
        <div>
          <div className="flex items-center gap-1.5">
            <h3 className="text-sm font-semibold">{t("Balances")}</h3>
            <InfoPopover title={t("Balances")}>
              <p>
                {t("Kept in days and built from the ledger: accruals post from the policy on its schedule, approved time off draws down, adjustments correct by hand. Available is the balance less requests still awaiting a decision.")}
              </p>
              <p>
                {t("A cap on the policy, or on the worker's tenure tier, stops accrual above it. A policy marked informational shows the figures without holding requests to them.")}
              </p>
            </InfoPopover>
          </div>
          <PolicyChip assignment={current} />
        </div>
        <div className="flex items-center gap-2">
          {canAssign ? (
            <Button size="sm" variant="outline" onClick={() => setAssignOpen(true)}>
              <ShieldCheckIcon className="size-3.5" />
              {current ? t("Change Policy") : t("Assign Policy")}
            </Button>
          ) : null}
          {canManage && current ? (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button size="sm" variant="ghost" className="size-8" aria-label={t("Balance actions")}>
                    <EllipsisIcon />
                  </Button>
                }
              />
              <DropdownMenuContent side="bottom" align="end">
                <DropdownMenuGroup>
                  <DropdownMenuLabel>{t("Manage")}</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    title={t("Adjust balance")}
                    description={t("Post a manual correction")}
                    onClick={() => setAdjustOpen(true)}
                  />
                  <DropdownMenuItem
                    title={t("Run accrual now")}
                    description={t("Post anything due since the last run")}
                    onClick={() => accrual.mutate()}
                  />
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </div>
      </div>

      {current ? (
        <BalanceCards balances={balances} />
      ) : (
        <div className="rounded-lg border border-dashed p-5 text-center">
          <ScaleIcon className="text-muted-foreground mx-auto size-6" />
          <p className="mt-2 text-sm font-medium">{t("No PTO policy assigned")}</p>
          <p className="text-muted-foreground mx-auto mt-1 max-w-md text-xs">
            {t("Time off is not tracked against a balance until this worker is enrolled in a policy.")}
          </p>
        </div>
      )}

      {current ? <LedgerTable workerId={workerId} /> : null}

      {assignments.length > 1 ? <AssignmentHistory assignments={assignments} /> : null}

      <AssignPolicyDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        workerId={workerId}
        currentPolicyId={current?.ptoPolicyId}
        onAssigned={() => void invalidate()}
      />
      <AdjustBalanceDialog
        open={adjustOpen}
        onOpenChange={setAdjustOpen}
        workerId={workerId}
        balances={balances}
        onAdjusted={() => void invalidate()}
      />
    </div>
  );
}

function PolicyChip({ assignment }: { assignment: PTOPolicyAssignment | null }) {
  const t = useT();

  if (!assignment?.ptoPolicy) {
    return <p className="text-muted-foreground text-xs">{t("Not enrolled in a policy.")}</p>;
  }
  return (
    <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
      <Badge variant="secondary" className="px-1.5 py-0 text-[10px]">
        {assignment.ptoPolicy.code}
      </Badge>
      {t("{0} · since {1} {2}", assignment.ptoPolicy.name, formatUnixDateMedium(assignment.effectiveFrom), !assignment.ptoPolicy.enforceBalance ? t("· informational") : "")}
    </p>
  );
}

function BalanceCards({ balances }: { balances: WorkerPTOBalanceView[] }) {
  const t = useT();

  const tracked = balances.filter((balance) => balance.tracked);
  if (tracked.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("The assigned policy does not track any PTO type yet.")}
      </p>
    );
  }
  return (
    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
      {tracked.map((balance) => (
        <BalanceCard key={balance.ptoType} balance={balance} />
      ))}
    </div>
  );
}

function BalanceCard({ balance }: { balance: WorkerPTOBalanceView }) {
  const t = useT();

  const max = balance.maxBalanceDays ? Number(balance.maxBalanceDays) : null;
  const ratio = max && max > 0 ? Math.min(1, Math.max(0, Number(balance.balanceDays) / max)) : null;
  const available = Number(balance.availableDays);

  return (
    <div
      className="bg-muted/30 rounded-lg border p-3"
      data-testid={`pto-balance-${balance.ptoType}`}
    >
      <div className="flex items-center justify-between">
        <p className="text-muted-foreground text-[11px] font-medium uppercase">
          {typeLabel(balance.ptoType)}
        </p>
        {!balance.enforced ? (
          <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
            {t("Not enforced")}
          </Badge>
        ) : null}
      </div>
      <p
        className={cn(
          "mt-1 text-xl font-semibold tabular-nums",
          available < 0 && "text-destructive",
        )}
      >
        {formatPtoDays(balance.availableDays)}
        <span className="text-muted-foreground ml-1 text-xs font-normal">available</span>
      </p>
      <dl className="text-muted-foreground mt-2 grid grid-cols-2 gap-x-3 gap-y-0.5 text-[11px]">
        <dt>{t("Balance")}</dt>
        <dd className="text-right tabular-nums">{formatPtoDays(balance.balanceDays)}</dd>
        <dt>{t("Pending")}</dt>
        <dd className="text-right tabular-nums">{formatPtoDays(balance.pendingDays)}</dd>
        <dt>{t("Accrued YTD")}</dt>
        <dd className="text-right tabular-nums">{formatPtoDays(balance.accruedYtdDays)}</dd>
        <dt>{t("Used YTD")}</dt>
        <dd className="text-right tabular-nums">{formatPtoDays(balance.usedYtdDays)}</dd>
      </dl>
      {ratio !== null ? (
        <div className="mt-2">
          <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
            <div className="bg-primary h-full rounded-full" style={{ width: `${ratio * 100}%` }} />
          </div>
          <p className="text-muted-foreground mt-0.5 text-[10px]">
            {t("Cap {0} days", formatPtoDays(balance.maxBalanceDays ?? "0"))}
          </p>
        </div>
      ) : null}
      {balance.nextAccrual ? (
        <p className="text-muted-foreground mt-2 flex items-center gap-1 text-[11px]">
          <CalendarSyncIcon className="size-3" />{t("+{0} on {1}", formatPtoDays(balance.nextAccrual.nominalDays), formatUnixDateMedium(balance.nextAccrual.effectiveAt))}
        </p>
      ) : null}
    </div>
  );
}

function entryTone(entry: WorkerPTOLedgerEntry): string {
  if (Number(entry.amountDays) < 0) return "text-destructive";
  if (entry.entryType === "Adjustment") return "text-warning";
  return "text-success";
}

function LedgerTable({ workerId }: { workerId: string }) {
  const t = useT();

  const [cursor, setCursor] = useState<string | null>(null);
  const [pages, setPages] = useState<WorkerPTOLedgerEntry[][]>([]);

  const query = useQuery({
    queryKey: [WORKER_PTO_LEDGER_KEY, workerId, cursor],
    queryFn: ({ signal }) =>
      fetchWorkerPtoLedger(
        {
          workerId,
          first: LEDGER_PAGE_SIZE,
          after: cursor ?? undefined,
          sort: [{ field: "effectiveAt", direction: "desc" }],
        },
        { signal },
      ),
  });

  const entries = useMemo(() => {
    const seen = new Set<string>();
    const merged: WorkerPTOLedgerEntry[] = [];
    for (const page of [...pages, query.data?.entries ?? []]) {
      for (const entry of page) {
        if (!seen.has(entry.id)) {
          seen.add(entry.id);
          merged.push(entry);
        }
      }
    }
    return merged;
  }, [pages, query.data]);

  const loadMore = () => {
    if (!query.data?.hasNextPage || !query.data.endCursor) return;
    setPages((prev) => [...prev, query.data?.entries ?? []]);
    setCursor(query.data.endCursor);
  };

  if (query.isLoading && entries.length === 0) {
    return <Skeleton className="h-32 w-full" />;
  }

  return (
    <div className="flex flex-col gap-2">
      <h4 className="text-sm font-semibold">{t("Ledger")}</h4>
      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-xs">
          <thead className="bg-muted/50">
            <tr>
              <th className="px-3 py-2 text-left font-medium">{t("Date")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("Type")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("Entry")}</th>
              <th className="px-3 py-2 text-right font-medium">{t("Days")}</th>
              <th className="px-3 py-2 text-right font-medium">{t("Balance")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("Note")}</th>
            </tr>
          </thead>
          <tbody>
            {entries.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-muted-foreground px-3 py-6 text-center">
                  {t("No ledger entries yet. Accruals post nightly from the effective date.")}
                </td>
              </tr>
            ) : (
              entries.map((entry) => (
                <tr key={entry.id} className="border-t">
                  <td className="px-3 py-2 tabular-nums">
                    {formatUnixDateMedium(entry.effectiveAt)}
                  </td>
                  <td className="px-3 py-2">{typeLabel(entry.ptoType)}</td>
                  <td className="px-3 py-2">
                    {PTO_LEDGER_ENTRY_LABELS[entry.entryType]}
                    {entry.periodKey ? (
                      <span className="text-muted-foreground ml-1 text-[10px]">
                        {entry.periodKey}
                      </span>
                    ) : null}
                  </td>
                  <td className={cn("px-3 py-2 text-right tabular-nums", entryTone(entry))}>
                    {Number(entry.amountDays) > 0 ? "+" : ""}
                    {formatPtoDays(entry.amountDays)}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPtoDays(entry.balanceAfterDays)}
                  </td>
                  <td
                    className="text-muted-foreground max-w-[220px] truncate px-3 py-2"
                    title={entry.note ?? undefined}
                  >
                    {entry.note ?? (entry.actorType === "System" ? t("Nightly accrual") : "—")}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {query.data?.hasNextPage ? (
        <Button size="sm" variant="ghost" className="self-center" onClick={loadMore}>
          {t("Load more")}
        </Button>
      ) : null}
    </div>
  );
}

function AssignmentHistory({ assignments }: { assignments: PTOPolicyAssignment[] }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-1">
      <h4 className="text-sm font-semibold">{t("Policy history")}</h4>
      <ul className="text-muted-foreground flex flex-col gap-0.5 text-xs">
        {assignments.map((assignment) => (
          <li key={assignment.id} className="flex items-center gap-2">
            <Badge variant="secondary" className="px-1.5 py-0 text-[10px]">
              {assignment.ptoPolicy?.code ?? "—"}
            </Badge>
            <span>
              {formatUnixDateMedium(assignment.effectiveFrom)} –{" "}
              {assignment.effectiveTo ? formatUnixDateMedium(assignment.effectiveTo) : "present"}
            </span>
            {assignment.note ? <span className="truncate">· {assignment.note}</span> : null}
          </li>
        ))}
      </ul>
    </div>
  );
}
