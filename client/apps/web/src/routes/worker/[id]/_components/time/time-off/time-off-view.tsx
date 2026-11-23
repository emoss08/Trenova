import { SectionPanel } from "@/components/section-panel";
import { VirtualRows } from "@/components/virtual-rows";
import { lazyWorkPane, preloadOn, useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerPto,
  fetchWorkerPtoBalances,
  fetchWorkerPtoPolicyAssignments,
  fetchWorkerPtoRequests,
  runPtoAccrual,
  WORKER_PTO_ASSIGNMENTS_KEY,
  WORKER_PTO_BALANCES_KEY,
  WORKER_PTO_REQUESTS_KEY,
  type PTOPolicyAssignment,
} from "@/lib/graphql/pto-policy";
import { PTOReasonDialog } from "@/routes/worker/_components/pto/pto-reason-dialog";
import { WorkerPTOBalances } from "@/routes/worker/_components/pto/worker-pto-balances";
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
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ListMoreLine, type ListMore } from "../../list-more";
import { useRecordById } from "../../use-record-by-id";
import { useRowFlash } from "../../use-row-flash";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { useWorkerFormGate } from "../../use-worker-form-gate";
import { useWorkerList } from "../../use-worker-list";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import { TimeSkeleton } from "../time-skeleton";
import { LedgerPanel } from "./ledger-panel";
import { RequestsPanel, type RequestAction } from "./requests-panel";

/** The requests read a page at a time; the server's most, so most workers need one. */
const REQUESTS_PAGE_SIZE = 100;

const TimeOffPane = lazyWorkPane(
  WORKER_FORM_PANES["request-time-off"],
  (module) => module.TimeOffPane,
  {
    openKey: (props) => props.pto?.id ?? "new",
    // A recorded request's pane is an edit or a read by its status; its own title says which.
    title: (props, t) => (props.pto ? undefined : t("Request time off")),
  },
);
const ChangePolicyPane = lazyWorkPane(
  WORKER_FORM_PANES["change-policy"],
  (module) => module.ChangePolicyPane,
  {
    title: (props, t) => (props.currentPolicyId ? t("Change policy") : t("Enroll in a policy")),
    fields: 3,
  },
);
const AdjustBalancePane = lazyWorkPane(
  WORKER_FORM_PANES["adjust-balance"],
  (module) => module.AdjustBalancePane,
  { title: (_props, t) => t("Adjust balance"), fields: 4 },
);

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
  const refresh = useWorkerInvalidation(workerId, "time-off");
  const [reason, setReason] = useState<RequestAction | null>(null);
  const [confirmAccrual, setConfirmAccrual] = useState(false);
  const { flash, flashOf } = useRowFlash();

  const balancesQuery = useQuery({
    queryKey: [WORKER_PTO_BALANCES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPtoBalances(workerId, { signal }),
  });
  const assignments = useWorkerList({
    root: WORKER_PTO_ASSIGNMENTS_KEY,
    workerId,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerPtoPolicyAssignments({ workerId, first, after }, { signal }),
  });
  const requests = useWorkerList({
    root: WORKER_PTO_REQUESTS_KEY,
    workerId,
    pageSize: REQUESTS_PAGE_SIZE,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerPtoRequests({ workerId, first, after }, { signal }),
  });
  // The request the address names: among the pages read, or past them, read by id.
  const requestRecord = useRecordById({
    root: WORKER_PTO_REQUESTS_KEY,
    workerId,
    id: open.timeOff,
    rows: requests.rows,
    rowsSettled: !requests.isPending,
    fetchRecord: fetchWorkerPto,
  });

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

  // The policy in force decides whether a balance can be adjusted.
  const policyInForce = assignments.rows.some((assignment) => assignment.effectiveTo == null);
  const opens = useWorkerFormGate(open.form, {
    "request-time-off": canCreate,
    "change-policy": canAssign,
    "adjust-balance": assignments.isPending ? "waiting" : canManage && policyInForce,
  });

  if (balancesQuery.isPending || assignments.isPending || requests.isPending) {
    return <TimeSkeleton view="time-off" />;
  }
  if (balancesQuery.isError || assignments.isError || requests.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={balancesQuery.error ?? assignments.error ?? requests.error}
        onRetry={() => {
          if (balancesQuery.isError) void balancesQuery.refetch();
          if (assignments.isError) assignments.refetch();
          if (requests.isError) requests.refetch();
        }}
        title={t("The time off could not be loaded")}
      />
    );
  }

  const balances = balancesQuery.data ?? [];
  const entries = requests.rows;
  // Newest first, so the policy in force (no end) is on the first page.
  const current = assignments.rows.find((assignment) => assignment.effectiveTo == null) ?? null;
  const record = requestRecord.record;
  const hasMenu = canAssign || (canManage && current !== null);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <PolicyLine assignment={current} />
        <div className="flex items-center gap-2">
          {canCreate ? (
            <Button size="sm" {...preloadOn(TimeOffPane)} onClick={() => go("request-time-off")}>
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
                      {...preloadOn(ChangePolicyPane)}
                      startContent={<SwitchHorizontal01Icon />}
                      title={current ? t("Change policy") : t("Enroll in a policy")}
                      onClick={() => go("change-policy")}
                    />
                  ) : null}
                  {canManage && current ? (
                    <DropdownMenuItem
                      {...preloadOn(AdjustBalancePane)}
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
                <Button
                  size="sm"
                  variant="secondary"
                  {...preloadOn(ChangePolicyPane)}
                  onClick={() => go("change-policy")}
                >
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
          total={requests.total}
          more={requests}
          onOpenRequest={(id) =>
            pane.closeThen(() => onOpen({ tab: "time", view: "time-off", timeOff: id }))
          }
          requestPane={TimeOffPane}
          onReason={setReason}
          onDecided={refresh}
          flashOf={flashOf}
          onChanged={flash}
        />
        {current ? <LedgerPanel workerId={workerId} /> : null}
      </div>

      {assignments.rows.length > 1 || assignments.hasMore ? (
        <PolicyHistory assignments={assignments.rows} more={assignments} />
      ) : null}

      <TimeOffPane.Component
        open={opens("request-time-off") || record !== null}
        onOpenChange={closePane}
        workerId={workerId}
        pto={open.form === "request-time-off" ? null : record}
        onSaved={(saved) => {
          if (saved.id) flash(saved.id);
          void refresh();
        }}
      />
      {canAssign ? (
        <ChangePolicyPane.Component
          open={opens("change-policy")}
          onOpenChange={closePane}
          workerId={workerId}
          currentPolicyId={current?.ptoPolicyId ?? null}
          onAssigned={refresh}
        />
      ) : null}
      {canManage && current ? (
        <AdjustBalancePane.Component
          open={opens("adjust-balance")}
          onOpenChange={closePane}
          workerId={workerId}
          balances={balances}
          onAdjusted={refresh}
        />
      ) : null}
      <PTOReasonDialog
        open={reason !== null}
        onOpenChange={(next) => {
          if (!next) setReason(null);
        }}
        ptoIds={reason?.pto.id ? [reason.pto.id] : []}
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

function PolicyHistory({
  assignments,
  more,
}: {
  assignments: readonly PTOPolicyAssignment[];
  more: ListMore;
}) {
  const t = useT();
  return (
    <SectionPanel title={t("Policy history")} count={more.hasMore ? undefined : assignments.length}>
      <VirtualRows
        rows={assignments.map((assignment) => ({
          key: assignment.id,
          render: () => <PolicyHistoryRow assignment={assignment} />,
        }))}
        aria-label={t("Policy history")}
        mask="card"
        className="max-h-56"
        contentClassName="px-1.5 py-1"
        hoverGlide
        estimateSize={36}
        onEndReached={more.hasMore && !more.loadMoreFailed ? more.loadMore : undefined}
        footer={<ListMoreLine more={more} />}
      />
    </SectionPanel>
  );
}

function PolicyHistoryRow({ assignment }: { assignment: PTOPolicyAssignment }) {
  const t = useT();
  return (
    <div
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
    </div>
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
