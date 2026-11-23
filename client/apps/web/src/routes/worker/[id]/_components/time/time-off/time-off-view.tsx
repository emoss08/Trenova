import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel } from "@/components/section-panel";
import { useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerPtoBalances,
  fetchWorkerPtoPolicyAssignments,
  runPtoAccrual,
  WORKER_PTO_ASSIGNMENTS_KEY,
  WORKER_PTO_BALANCES_KEY,
  type PTOPolicyAssignment,
} from "@/lib/graphql/pto-policy";
import { worker as workerQueries } from "@/lib/queries/worker";
import { PTOReasonDialog } from "@/routes/worker/_components/pto/pto-reason-dialog";
import { usePTOInvalidation } from "@/routes/worker/_components/pto/use-pto-invalidation";
import {
  useWorkerPtoInvalidation,
  WorkerPTOBalances,
} from "@/routes/worker/_components/pto/worker-pto-balances";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  DotsHorizontalIcon,
  PlusIcon,
  RefreshCw01Icon,
  Sliders01Icon,
  SwitchHorizontal01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { TimeSkeleton } from "../time-skeleton";
import { AdjustBalancePane } from "./adjust-balance-pane";
import { ChangePolicyPane } from "./change-policy-pane";
import { LedgerPanel } from "./ledger-panel";
import { RequestsPanel, type RequestAction } from "./requests-panel";
import { TimeOffPane } from "./request-time-off-pane";

type TimeOffViewProps = Pick<WorkerTabProps, "workerId" | "worker" | "open" | "onOpen">;

/**
 * The worker's time off: the policy they are on, what each balance has free against its cap, the
 * requests with their decisions, and the ledger behind the figures.
 */
export default function TimeOffView({ workerId, worker, open, onOpen }: TimeOffViewProps) {
  const t = useT();
  const pane = useWorkPane();
  const { allowed: canCreate } = usePermission(Resource.WorkerPTO, Operation.Create);
  const { allowed: canAssign } = usePermission(Resource.PTOPolicy, Operation.Assign);
  const { allowed: canManage } = usePermission(Resource.WorkerPTO, Operation.Manage);
  const invalidateWorker = useWorkerPtoInvalidation(workerId);
  const invalidateRoster = usePTOInvalidation();
  const [reason, setReason] = useState<RequestAction | null>(null);
  const [changed, setChanged] = useState<{ id: string; tick: "a" | "b" } | null>(null);
  const [confirmAccrual, setConfirmAccrual] = useState(false);
  const reasonIds = useMemo(() => (reason?.pto.id ? [reason.pto.id] : []), [reason]);

  const balancesQuery = useQuery({
    queryKey: [WORKER_PTO_BALANCES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPtoBalances(workerId, { signal }),
  });
  const assignmentsQuery = useQuery({
    queryKey: [WORKER_PTO_ASSIGNMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPtoPolicyAssignments(workerId, { signal }),
  });
  const historyQuery = useQuery(workerQueries.ptoHistory(workerId));

  const refresh = () => Promise.all([invalidateWorker(), invalidateRoster()]);
  const flash = (id: string) =>
    setChanged((previous) => ({ id, tick: previous?.tick === "a" ? "b" : "a" }));

  const accrual = useApiMutation({
    resourceName: "Accrual",
    scope: { id: `worker-pto-accrual:${workerId}` },
    mutationFn: () => runPtoAccrual({ workerId }),
    onSuccess: async (result) => {
      await refresh();
      toast.success(result.entriesPosted > 0 ? t("Accrual posted") : t("Nothing to accrue yet"), {
        description: t(
          "{0} posted, {1} held at the cap, {2} already posted.",
          result.entriesPosted,
          result.entriesCapped,
          result.entriesSkipped,
        ),
      });
    },
  });

  const go = (form: "request-time-off" | "change-policy" | "adjust-balance") =>
    pane.closeThen(() => onOpen({ tab: "time", view: "time-off", form }));
  const closePane = (next: boolean) => {
    if (!next) onOpen({ tab: "time", view: "time-off" });
  };

  if (balancesQuery.isPending || assignmentsQuery.isPending || historyQuery.isPending) {
    return <TimeSkeleton view="time-off" />;
  }
  const failed = [balancesQuery, assignmentsQuery, historyQuery].find((query) => query.isError);
  if (failed) {
    return (
      <ErrorState
        layout="sheet"
        error={failed.error}
        onRetry={() => {
          void balancesQuery.refetch();
          void assignmentsQuery.refetch();
          void historyQuery.refetch();
        }}
        title={t("Time off could not be loaded")}
      />
    );
  }

  const balances = balancesQuery.data ?? [];
  const assignments = assignmentsQuery.data ?? [];
  const entries = historyQuery.data ?? [];
  const current = assignments.find((assignment) => assignment.effectiveTo == null) ?? null;
  const record = open.timeOff ? (entries.find((entry) => entry.id === open.timeOff) ?? null) : null;
  const hasMenu = canAssign || (canManage && current !== null);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <PolicyLine assignment={current} />
        <div className="flex items-center gap-2">
          {canCreate ? (
            <Button size="sm" onClick={() => go("request-time-off")}>
              <PlusIcon className="size-3.5" />
              {t("Request time off")}
            </Button>
          ) : null}
          {hasMenu ? (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button size="icon-sm" variant="outline" aria-label={t("Time off options")}>
                    <DotsHorizontalIcon />
                  </Button>
                }
              />
              <DropdownMenuContent side="bottom" align="end">
                <DropdownMenuGroup>
                  {canAssign ? (
                    <DropdownMenuItem
                      startContent={<SwitchHorizontal01Icon />}
                      title={current ? t("Change policy") : t("Enroll in a policy")}
                      onClick={() => go("change-policy")}
                    />
                  ) : null}
                  {canManage && current ? (
                    <DropdownMenuItem
                      startContent={<Sliders01Icon />}
                      title={t("Adjust balance")}
                      onClick={() => go("adjust-balance")}
                    />
                  ) : null}
                </DropdownMenuGroup>
                {canManage && current ? (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuGroup>
                      <DropdownMenuItem
                        startContent={<RefreshCw01Icon />}
                        title={t("Run accrual now")}
                        onClick={() => setConfirmAccrual(true)}
                      />
                    </DropdownMenuGroup>
                  </>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </div>
      </div>

      {current ? (
        <WorkerPTOBalances balances={balances} />
      ) : (
        <div className="rounded-lg border bg-card">
          <EmptySheet
            sketch={<BalancesSketch />}
            title={t("Not enrolled in a time off policy")}
            description={t(
              "Time off is not counted against a balance until {0} is on a policy. Requests can still be recorded.",
              worker.wholeName,
            )}
            action={
              canAssign ? (
                <Button size="sm" variant="secondary" onClick={() => go("change-policy")}>
                  {t("Enroll in a policy")}
                </Button>
              ) : undefined
            }
          />
        </div>
      )}

      <div className="grid min-w-0 items-start gap-4 lg:grid-cols-2">
        <RequestsPanel
          entries={entries}
          onOpenRequest={(id) =>
            pane.closeThen(() => onOpen({ tab: "time", view: "time-off", timeOff: id }))
          }
          onReason={setReason}
          onDecided={refresh}
          changed={changed}
          onChanged={flash}
        />
        {current ? <LedgerPanel workerId={workerId} /> : null}
      </div>

      {assignments.length > 1 ? <PolicyHistory assignments={assignments} /> : null}

      <TimeOffPane
        open={open.form === "request-time-off" || record !== null}
        onOpenChange={closePane}
        workerId={workerId}
        pto={open.form === "request-time-off" ? null : record}
        onSaved={(saved) => {
          if (saved.id) flash(saved.id);
          void refresh();
        }}
      />
      {canAssign ? (
        <ChangePolicyPane
          open={open.form === "change-policy"}
          onOpenChange={closePane}
          workerId={workerId}
          currentPolicyId={current?.ptoPolicyId ?? null}
          onAssigned={() => void refresh()}
        />
      ) : null}
      {canManage && current ? (
        <AdjustBalancePane
          open={open.form === "adjust-balance"}
          onOpenChange={closePane}
          workerId={workerId}
          balances={balances}
          onAdjusted={() => void refresh()}
        />
      ) : null}
      <PTOReasonDialog
        open={reason !== null}
        onOpenChange={(next) => {
          if (!next) setReason(null);
        }}
        ptoIds={reasonIds}
        mode={reason?.kind ?? "reject"}
        onCompleted={() => {
          if (reason?.pto.id) flash(reason.pto.id);
          void refresh();
        }}
      />
      <ConfirmDialog
        variant="default"
        open={confirmAccrual}
        onOpenChange={setConfirmAccrual}
        title={t("Run accrual now?")}
        description={t(
          "Posts every accrual due under {0}'s policy since the last run. Accruals already posted are skipped.",
          worker.wholeName,
        )}
        confirmLabel={t("Run accrual")}
        pending={accrual.isPending}
        onConfirm={() => accrual.mutateAsync()}
      />
    </div>
  );
}

function PolicyLine({ assignment }: { assignment: PTOPolicyAssignment | null }) {
  const t = useT();
  if (!assignment?.ptoPolicy) {
    return <p className="text-sm text-muted-foreground">{t("Not on a time off policy")}</p>;
  }
  return (
    <p className="flex flex-wrap items-center gap-2">
      <Badge variant="neutral" className="font-mono">
        {assignment.ptoPolicy.code}
      </Badge>
      <span className="text-muted-foreground tabular-nums">
        {assignment.ptoPolicy.enforceBalance
          ? t(
              "{0}, since {1}",
              assignment.ptoPolicy.name,
              formatUnixDateMedium(assignment.effectiveFrom),
            )
          : t(
              "{0}, since {1} · balances are not enforced",
              assignment.ptoPolicy.name,
              formatUnixDateMedium(assignment.effectiveFrom),
            )}
      </span>
    </p>
  );
}

function PolicyHistory({ assignments }: { assignments: readonly PTOPolicyAssignment[] }) {
  const t = useT();
  return (
    <SectionPanel title={t("Policy history")} count={assignments.length}>
      <ScrollArea className="max-h-56" maskVariant="card">
        <HoverGlideGroup>
          <ul className="flex flex-col px-1.5 py-1">
            {assignments.map((assignment) => (
              <li
                key={assignment.id}
                data-glide-target
                className="flex flex-wrap items-center gap-2 rounded-md px-1.5 py-2 text-xs"
              >
                <Badge variant="neutral" className="font-mono">
                  {assignment.ptoPolicy?.code ?? t("Policy removed")}
                </Badge>
                <span className="font-mono text-muted-foreground tabular-nums">
                  {assignment.effectiveTo
                    ? formatRange(assignment.effectiveFrom, assignment.effectiveTo, {
                        year: "always",
                        join: "words",
                      })
                    : t("{0} to now", formatUnixDateMedium(assignment.effectiveFrom))}
                </span>
                {assignment.note ? (
                  <span className="min-w-0 truncate text-muted-foreground">{assignment.note}</span>
                ) : null}
              </li>
            ))}
          </ul>
        </HoverGlideGroup>
      </ScrollArea>
    </SectionPanel>
  );
}

function BalancesSketch() {
  return (
    <div className="grid grid-cols-3 rounded-md border border-border/70 bg-card text-left">
      {[0, 1, 2].map((cell) => (
        <div key={cell} className="flex flex-col gap-2 border-border/60 p-3 not-first:border-l">
          <GhostLine className="w-1/3" />
          <GhostLine className="h-3 w-1/4" />
          <GhostLine className="w-full" />
        </div>
      ))}
    </div>
  );
}
