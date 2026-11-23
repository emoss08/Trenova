import { RowActionsMenu } from "@/components/row-actions-menu";
import { SectionPanel, SectionPanelLink, SectionPanelQuiet } from "@/components/section-panel";
import { useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  endWorkerPayAssignment,
  fetchCurrentSettlementPeriod,
  fetchEffectiveWorkerPayAssignment,
  fetchWorkerEarningsSummary,
  fetchWorkerPayAssignments,
  type EffectiveWorkerPayAssignment,
} from "@/lib/graphql/driver-settlement";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { SlashCircle01Icon, Wallet02Icon } from "@trenova/shared/components/icons";
import { PayeeClassificationBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import type { PayeeClassification } from "@trenova/shared/types/driver-pay";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PaySkeleton } from "../pay-skeleton";
import { formatPayRate, PAY_METHOD_UNITS, payComponentLines, sortAssignments } from "../pay-model";
import { ChangeProfilePane } from "./change-profile-pane";

const WORKER_PAY_KEY = "worker-pay";

type PayProfileViewProps = Pick<WorkerTabProps, "workerId" | "worker" | "open" | "onOpen">;

/**
 * The driver's pay: the profile in force with each component's rate and this driver's overrides,
 * the guaranteed minimum, every assignment before it, and what is owed and held. Changing the
 * profile opens beside it; ending the assignment asks first.
 */
export default function WorkerPayTab({ workerId, worker, open, onOpen }: PayProfileViewProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const { closeThen } = useWorkPane();
  const { allowed: canAssign } = usePermission(Resource.DriverPayProfile, Operation.Assign);
  const { allowed: canReadSettlements } = usePermission(Resource.DriverSettlement, Operation.Read);
  const [ending, setEnding] = useState(false);

  const assignmentQuery = useQuery({
    queryKey: [WORKER_PAY_KEY, "effective-assignment", workerId],
    queryFn: ({ signal }) => fetchEffectiveWorkerPayAssignment(workerId, { signal }),
  });
  const historyQuery = useQuery({
    queryKey: [WORKER_PAY_KEY, "assignment-history", workerId],
    queryFn: ({ signal }) => fetchWorkerPayAssignments(workerId, { signal }),
  });
  const earningsQuery = useQuery({
    queryKey: [WORKER_PAY_KEY, "earnings-summary", workerId],
    queryFn: ({ signal }) => fetchWorkerEarningsSummary(workerId, { signal }),
    enabled: canReadSettlements,
  });
  const periodQuery = useQuery({
    queryKey: ["settlement-period", "current"],
    queryFn: ({ signal }) => fetchCurrentSettlementPeriod({ signal }),
    enabled: canReadSettlements,
  });

  const invalidate = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [WORKER_PAY_KEY] }),
      queryClient.invalidateQueries({ queryKey: ["worker"] }),
    ]);

  const end = useApiMutation({
    scope: { id: `worker-pay-end:${workerId}` },
    resourceName: "Pay assignment",
    mutationFn: (assignmentId: string) =>
      endWorkerPayAssignment({ assignmentId, endDate: getTodayDate() }),
    onSuccess: () => void invalidate(),
  });

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
  const history = sortAssignments(historyQuery.data ?? []);
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
                sketch={
                  <div className="flex flex-col gap-2 rounded-md border bg-card p-4">
                    <GhostLine className="w-1/3" />
                    <GhostLine className="w-2/3" />
                    <GhostLine className="w-1/2" />
                  </div>
                }
                action={
                  canAssign ? (
                    <Button size="sm" onClick={openChange}>
                      <Wallet02Icon className="size-3.5" />
                      {t("Assign profile")}
                    </Button>
                  ) : undefined
                }
              />
            </SectionPanel>
          )}

          <SectionPanel title={t("Assignment history")} count={history.length}>
            {historyQuery.isPending ? (
              <div aria-busy aria-label={t("Loading the assignment history")} className="p-3">
                <span className="block h-3 w-1/2 ui-shimmer rounded-sm" />
              </div>
            ) : historyQuery.isError ? (
              <ErrorState
                error={historyQuery.error}
                layout="compact"
                title={t("The assignment history could not be loaded")}
                onRetry={() => void historyQuery.refetch()}
              />
            ) : history.length === 0 ? (
              <SectionPanelQuiet>{t("No assignments yet.")}</SectionPanelQuiet>
            ) : (
              <ScrollArea viewportClassName="max-h-64" maskVariant="card">
                <table className="w-full min-w-[32rem] text-sm">
                  <thead className="sticky top-0 z-10 bg-card text-left text-xs text-muted-foreground">
                    <tr className="border-b border-border-subtle">
                      <th className="px-3 py-2 font-medium">{t("Profile")}</th>
                      <th className="px-3 py-2 font-medium">{t("Effective")}</th>
                      <th className="px-3 py-2 font-medium">{t("Ended")}</th>
                      <th className="px-3 py-2 text-right font-medium">{t("Split")}</th>
                      <th className="px-3 py-2 font-medium">{t("Overrides")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((row) => {
                      const current = assignment?.id === row.id;
                      const overrides = row.rateOverrides?.length ?? 0;
                      return (
                        <tr key={row.id} className="border-b border-border-subtle last:border-0">
                          <td className="px-3 py-2">
                            <span className="font-medium">{row.payProfile?.name ?? "—"}</span>
                            {current ? (
                              <Badge variant="success" className="ml-2">
                                {t("Current")}
                              </Badge>
                            ) : null}
                          </td>
                          <td className="px-3 py-2 font-mono text-xs">
                            {formatUnixDateMedium(row.effectiveFrom)}
                          </td>
                          <td
                            className={cn(
                              "px-3 py-2 text-xs",
                              row.effectiveTo ? "font-mono" : "text-muted-foreground",
                            )}
                          >
                            {row.effectiveTo
                              ? formatUnixDateMedium(row.effectiveTo)
                              : t("Still in effect")}
                          </td>
                          <td className="px-3 py-2 text-right font-mono text-xs">
                            {t("{0}%", Number(row.splitPercent))}
                          </td>
                          <td
                            className={cn("px-3 py-2", overrides === 0 && "text-muted-foreground")}
                          >
                            {overrides === 0
                              ? t("None")
                              : t("{0, plural, one {# override} other {# overrides}}", overrides)}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </ScrollArea>
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
              <ErrorState
                error={earningsQuery.error}
                layout="compact"
                title={t("What is owed could not be loaded")}
                onRetry={() => void earningsQuery.refetch()}
              />
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

      <ChangeProfilePane
        open={open.form === "assign-pay-profile" && canAssign}
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
            <Button size="sm" variant="secondary" onClick={onChange}>
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
        <ScrollArea maskVariant="card">
          <table className="w-full min-w-[26rem] text-sm">
            <thead className="text-left text-xs text-muted-foreground">
              <tr className="border-b border-border-subtle">
                <th className="px-4 py-2 font-medium">{t("Component")}</th>
                <th className="px-3 py-2 text-right font-medium">{t("Profile rate")}</th>
                <th className="px-3 py-2 text-right font-medium">{t("This driver")}</th>
              </tr>
            </thead>
            <tbody>
              {lines.map((line) => {
                const unit = t(PAY_METHOD_UNITS[line.method] ?? "");
                return (
                  <tr key={line.id} className="border-b border-border-subtle last:border-0">
                    <td className="px-4 py-2">
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
                    <td className="px-3 py-2 text-right">
                      <span className="font-mono">
                        {formatPayRate(line.method, line.profileRate)}
                      </span>{" "}
                      <span className="text-xs text-muted-foreground">{unit}</span>
                    </td>
                    <td
                      data-override={line.override !== null ? "" : undefined}
                      className={cn(
                        "px-3 py-2 text-right",
                        line.override !== null
                          ? "bg-brand-subtle font-medium"
                          : "text-muted-foreground",
                      )}
                    >
                      <span className="font-mono">
                        {formatPayRate(line.method, line.override ?? line.profileRate)}
                      </span>{" "}
                      <span className="text-xs font-normal text-muted-foreground">{unit}</span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </ScrollArea>
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
