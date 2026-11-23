import { RowActionsMenu } from "@/components/row-actions-menu";
import { SectionPanel, SectionPanelLink, SectionPanelQuiet } from "@/components/section-panel";
import { lazyWorkPane, preloadOn, useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  endWorkerPayAssignment,
  fetchCurrentSettlementPeriod,
  fetchEffectiveWorkerPayAssignment,
  fetchWorkerEarningsSummary,
  fetchWorkerPayAssignments,
  type EffectiveWorkerPayAssignment,
  type WorkerPayAssignmentRow,
} from "@/lib/graphql/driver-settlement";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { SlashCircle01Icon, Wallet02Icon } from "@trenova/shared/components/icons";
import { PayeeClassificationBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import type { PayeeClassification } from "@trenova/shared/types/driver-pay";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import {
  BOUNDED_CELL_CLASS,
  BoundedTable,
  type BoundedColumn,
  type BoundedRowProps,
} from "../../bounded-table";
import { LineSketch } from "../../list-sketch";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { useWorkerFormGate } from "../../use-worker-form-gate";
import { useWorkerList } from "../../use-worker-list";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import { formatPayRate, PAY_METHOD_UNITS, payComponentLines } from "../pay-model";
import { PaySkeleton } from "../pay-skeleton";

export const WORKER_PAY_KEY = "worker-pay";

const ChangeProfilePane = lazyWorkPane(
  WORKER_FORM_PANES["assign-pay-profile"],
  (module) => module.ChangeProfilePane,
  {
    title: (props, t) => (props.current ? t("Change pay profile") : t("Assign pay profile")),
    fields: 4,
  },
);

type PayProfileViewProps = Pick<WorkerTabProps, "workerId" | "worker" | "open" | "onOpen">;

/**
 * The driver's pay: the profile in force with each component's rate and this driver's overrides,
 * the guaranteed minimum, every assignment before it, and what is owed and held. Changing the
 * profile opens beside it; ending the assignment asks first.
 */
export default function WorkerPayTab({ workerId, worker, open, onOpen }: PayProfileViewProps) {
  const t = useT();
  const { closeThen } = useWorkPane();
  const { allowed: canAssign } = usePermission(Resource.DriverPayProfile, Operation.Assign);
  const { allowed: canReadSettlements } = usePermission(Resource.DriverSettlement, Operation.Read);
  const [ending, setEnding] = useState(false);

  const assignmentQuery = useQuery({
    queryKey: [WORKER_PAY_KEY, workerId, "effective-assignment"],
    queryFn: ({ signal }) => fetchEffectiveWorkerPayAssignment(workerId, { signal }),
  });
  const history = useWorkerList({
    root: WORKER_PAY_KEY,
    workerId,
    pageSize: 100,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerPayAssignments({ workerId, first, after }, { signal }),
  });
  const earningsQuery = useQuery({
    queryKey: [WORKER_PAY_KEY, workerId, "earnings-summary"],
    queryFn: ({ signal }) => fetchWorkerEarningsSummary(workerId, { signal }),
    enabled: canReadSettlements,
  });
  const periodQuery = useQuery({
    queryKey: ["settlement-period", "current"],
    queryFn: ({ signal }) => fetchCurrentSettlementPeriod({ signal }),
    enabled: canReadSettlements,
  });

  const invalidate = useWorkerInvalidation(workerId, "pay-profile");

  const end = useApiMutation({
    scope: { id: `worker-pay-end:${workerId}` },
    resourceName: "Pay assignment",
    mutationFn: (assignmentId: string) =>
      endWorkerPayAssignment({ assignmentId, endDate: getTodayDate() }),
    onSuccess: () => invalidate(),
  });

  const opens = useWorkerFormGate(open.form, { "assign-pay-profile": canAssign });

  if (assignmentQuery.isPending) return <PaySkeleton view="pay-profile" />;
  if (assignmentQuery.isError) {
    return (
      <ErrorState
        error={assignmentQuery.error}
        layout="sheet"
        title={t("The pay profile could not be loaded")}
        onRetry={() => void assignmentQuery.refetch()}
      />
    );
  }

  const assignment = assignmentQuery.data ?? null;
  const openChange = () =>
    closeThen(() => onOpen({ tab: "pay", view: "pay-profile", form: "assign-pay-profile" }));
  const close = () => onOpen({ tab: "pay", view: "pay-profile" });

  return (
    <div className="@container text-sm">
      <div className="grid items-start gap-4 @5xl:grid-cols-[minmax(0,1fr)_24rem]">
        <div className="flex min-w-0 flex-col gap-4">
          {assignment ? (
            <CurrentProfile
              assignment={assignment}
              canAssign={canAssign}
              pending={end.isPending}
              onChange={openChange}
              onEnd={() => setEnding(true)}
            />
          ) : (
            <SectionPanel title={t("Pay profile")}>
              <EmptySheet
                title={t("No pay profile assigned")}
                description={t(
                  "{0} does not accrue pay for delivered shipments until a profile is assigned. Assign a shared profile and set this driver's own rates on it where they differ.",
                  worker.firstName,
                )}
                sketch={<LineSketch widths={["w-1/3", "w-2/3", "w-1/2"]} />}
                action={
                  canAssign ? (
                    <Button size="sm" {...preloadOn(ChangeProfilePane)} onClick={openChange}>
                      <Wallet02Icon className="size-3.5" />
                      {t("Assign profile")}
                    </Button>
                  ) : undefined
                }
              />
            </SectionPanel>
          )}

          <SectionPanel
            title={t("Assignment history")}
            count={history.total ?? (history.hasMore ? undefined : history.rows.length)}
          >
            {history.isPending ? (
              <div aria-busy aria-label={t("Loading the assignment history")} className="p-3">
                <span className="block h-3 w-1/2 ui-shimmer rounded-sm" />
              </div>
            ) : history.isError ? (
              <div className="p-3">
                <ErrorState
                  error={history.error}
                  layout="compact"
                  title={t("The assignment history could not be loaded")}
                  onRetry={history.refetch}
                />
              </div>
            ) : history.rows.length === 0 ? (
              <SectionPanelQuiet>{t("No assignments yet.")}</SectionPanelQuiet>
            ) : (
              <BoundedTable
                label={t("Assignment history")}
                columns={historyColumns(t)}
                maxHeightClassName="max-h-64"
                minWidthClassName="min-w-128"
                rows={history.rows.map((row) => ({
                  key: row.id,
                  render: (props) => (
                    <HistoryRow props={props} row={row} current={assignment?.id === row.id} />
                  ),
                }))}
                more={history}
              />
            )}
          </SectionPanel>
        </div>

        {canReadSettlements ? (
          <SectionPanel
            title={t("Owed and held")}
            action={
              <SectionPanelLink to="/payroll/settlements">{t("Open settlements")}</SectionPanelLink>
            }
          >
            {earningsQuery.isPending ? (
              <div
                aria-busy
                aria-label={t("Loading what is owed")}
                className="flex flex-col gap-2 p-3"
              >
                <span className="block h-3 w-3/4 ui-shimmer rounded-sm" />
                <span className="block h-3 w-1/2 ui-shimmer rounded-sm" />
              </div>
            ) : earningsQuery.isError ? (
              <div className="p-3">
                <ErrorState
                  error={earningsQuery.error}
                  layout="compact"
                  title={t("What is owed could not be loaded")}
                  onRetry={() => void earningsQuery.refetch()}
                />
              </div>
            ) : earningsQuery.data ? (
              <DescriptionList layout="split" className="px-3">
                <DescriptionItem label={t("Unsettled earnings")} numeric>
                  <span className="font-mono">
                    {formatMinor(earningsQuery.data.accruedGrossMinor)}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {t(
                      "{0, plural, one {# pay event} other {# pay events}}",
                      earningsQuery.data.accruedEventCount,
                    )}
                  </span>
                </DescriptionItem>
                <DescriptionItem label={t("Outstanding advances")} numeric>
                  <span className="font-mono">
                    {formatMinor(earningsQuery.data.outstandingAdvances)}
                  </span>
                  {earningsQuery.data.outstandingAdvances > 0 ? (
                    <span className="block text-xs text-muted-foreground">
                      {t("Taken from the next settlement")}
                    </span>
                  ) : null}
                </DescriptionItem>
                <DescriptionItem label={t("Escrow balance")} numeric>
                  <span className="font-mono">
                    {formatMinor(earningsQuery.data.escrowBalanceMinor)}
                  </span>
                </DescriptionItem>
                {periodQuery.data?.payDate ? (
                  <DescriptionItem label={t("Next settlement")} numeric>
                    <span className="font-mono">
                      {formatUnixDateMedium(periodQuery.data.payDate)}
                    </span>
                  </DescriptionItem>
                ) : null}
              </DescriptionList>
            ) : (
              <SectionPanelQuiet>{t("Nothing owed or held.")}</SectionPanelQuiet>
            )}
          </SectionPanel>
        ) : null}
      </div>

      <ChangeProfilePane.Component
        open={opens("assign-pay-profile")}
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        workerName={worker.firstName}
        current={assignment}
        onSaved={() => {
          void invalidate();
          close();
        }}
      />
      {assignment ? (
        <ConfirmDialog
          open={ending}
          onOpenChange={setEnding}
          title={t("End this pay assignment?")}
          description={t(
            "It ends today. {0} stops accruing pay for shipments delivered after today until a new profile is assigned; pay already accrued is kept.",
            worker.firstName,
          )}
          confirmLabel={t("End assignment")}
          pending={end.isPending}
          onConfirm={() => end.mutateAsync(assignment.id)}
        />
      ) : null}
    </div>
  );
}

function CurrentProfile({
  assignment,
  canAssign,
  pending,
  onChange,
  onEnd,
}: {
  assignment: EffectiveWorkerPayAssignment;
  canAssign: boolean;
  pending: boolean;
  onChange: () => void;
  onEnd: () => void;
}) {
  const t = useT();
  const profile = assignment.payProfile;
  const lines = payComponentLines(assignment);
  const minimum = profile?.guaranteedPeriodMinimumMinor ?? 0;

  return (
    <SectionPanel
      title={t("Pay profile")}
      action={
        canAssign ? (
          <>
            <Button
              size="sm"
              variant="secondary"
              {...preloadOn(ChangeProfilePane)}
              onClick={onChange}
            >
              {t("Change profile")}
            </Button>
            <RowActionsMenu
              label={t("Pay profile options")}
              actions={[
                {
                  id: "end",
                  label: t("End assignment"),
                  icon: SlashCircle01Icon,
                  destructive: true,
                  disabled: pending,
                  onSelect: onEnd,
                },
              ]}
            />
          </>
        ) : null
      }
    >
      <div className="border-b border-border-subtle py-3 pr-3 pl-4">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="text-base font-medium">{profile?.name ?? t("Pay profile")}</span>
          {profile ? (
            <PayeeClassificationBadge
              classification={profile.classification as PayeeClassification}
            />
          ) : null}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
          {t(
            "since {0} · {1}% split",
            formatUnixDateMedium(assignment.effectiveFrom),
            Number(assignment.splitPercent),
          )}
        </p>
      </div>
      {lines.length === 0 ? (
        <SectionPanelQuiet>{t("This profile has no active components.")}</SectionPanelQuiet>
      ) : (
        <BoundedTable
          label={t("Pay components")}
          columns={componentColumns(t)}
          maxHeightClassName="max-h-96"
          minWidthClassName="min-w-104"
        >
          {lines.map((line) => {
            const unit = t(PAY_METHOD_UNITS[line.method] ?? "");
            return (
              <tr key={line.id} data-glide-target>
                <td className={BOUNDED_CELL_CLASS}>
                  {line.name}
                  {line.bandCount > 0 ? (
                    <span className="block text-xs text-muted-foreground">
                      {t(
                        "{0, plural, one {# mileage band} other {# mileage bands}}",
                        line.bandCount,
                      )}
                    </span>
                  ) : null}
                </td>
                <td className={cn(BOUNDED_CELL_CLASS, "text-right whitespace-nowrap")}>
                  <span className="font-mono tabular-nums">
                    {formatPayRate(line.method, line.profileRate)}
                  </span>{" "}
                  <span className="text-xs text-muted-foreground">{unit}</span>
                </td>
                <td
                  data-override={line.override !== null ? "" : undefined}
                  className={cn(
                    BOUNDED_CELL_CLASS,
                    "text-right whitespace-nowrap",
                    line.override !== null
                      ? "bg-brand-subtle font-medium"
                      : "text-muted-foreground",
                  )}
                >
                  <span className="font-mono tabular-nums">
                    {formatPayRate(line.method, line.override ?? line.profileRate)}
                  </span>{" "}
                  <span className="text-xs font-normal text-muted-foreground">{unit}</span>
                </td>
              </tr>
            );
          })}
        </BoundedTable>
      )}
      {minimum > 0 ? (
        <div className="flex items-center justify-between gap-4 border-t border-border-subtle py-2.5 pr-3 pl-4">
          <span>
            {t("Guaranteed minimum a pay period")}
            <span className="block text-xs text-muted-foreground">
              {t("Tops up any period whose pay events come to less.")}
            </span>
          </span>
          <span className="font-mono font-medium">{formatMinor(minimum)}</span>
        </div>
      ) : null}
    </SectionPanel>
  );
}

function historyColumns(t: TranslateFn): BoundedColumn[] {
  return [
    { key: "profile", label: t("Profile") },
    { key: "effective", label: t("Effective") },
    { key: "ended", label: t("Ended") },
    { key: "split", label: t("Split"), align: "end" },
    { key: "overrides", label: t("Overrides") },
  ];
}

function componentColumns(t: TranslateFn): BoundedColumn[] {
  return [
    { key: "component", label: t("Component") },
    { key: "profile-rate", label: t("Profile rate"), align: "end" },
    { key: "this-driver", label: t("This driver"), align: "end" },
  ];
}

function HistoryRow({
  props,
  row,
  current,
}: {
  props: BoundedRowProps;
  row: WorkerPayAssignmentRow;
  current: boolean;
}) {
  const t = useT();
  const overrides = row.rateOverrides?.length ?? 0;
  return (
    <tr {...props} data-glide-target>
      <td className={BOUNDED_CELL_CLASS}>
        <span className="font-medium">{row.payProfile?.name ?? "—"}</span>
        {current ? (
          <Badge variant="success" dot className="ml-2">
            {t("Current")}
          </Badge>
        ) : null}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs tabular-nums")}>
        {formatUnixDateMedium(row.effectiveFrom)}
      </td>
      <td
        className={cn(
          BOUNDED_CELL_CLASS,
          "text-xs",
          row.effectiveTo ? "font-mono tabular-nums" : "text-muted-foreground",
        )}
      >
        {row.effectiveTo ? formatUnixDateMedium(row.effectiveTo) : t("Still in effect")}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "text-right font-mono text-xs tabular-nums")}>
        {t("{0}%", Number(row.splitPercent))}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, overrides === 0 && "text-muted-foreground")}>
        {overrides === 0
          ? t("None")
          : t("{0, plural, one {# override} other {# overrides}}", overrides)}
      </td>
    </tr>
  );
}
