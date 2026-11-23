import { HoverGlideGroup } from "@/components/hover-glide";
import { ReasonDialog } from "@/components/reason-dialog";
import { RowActionsMenu } from "@/components/row-actions-menu";
import { SectionPanel } from "@/components/section-panel";
import { useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  BENEFIT_ENROLLMENTS_KEY,
  endBenefitEnrollment,
  fetchWorkerBenefitEnrollments,
  fetchWorkerTotalCompensation,
  TOTAL_COMPENSATION_KEY,
  type BenefitEnrollmentRow,
} from "@/lib/graphql/benefits";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Edit02Icon, PlusIcon, SlashCircle01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  benefitPlanTypeLabel,
  coverageTierLabel,
  enrollmentStatusLabel,
  enrollmentStatusTone,
} from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { PaySkeleton } from "../pay-skeleton";
import { CompensationBar } from "./compensation-bar";
import { EnrollPane } from "./enroll-pane";

type BenefitsViewProps = Pick<WorkerTabProps, "workerId" | "open" | "onOpen">;

/** Covered first, then elections still to make, then declines, then ended cover. */
const STATUS_ORDER: Record<string, number> = { Active: 0, Pending: 1, Waived: 2, Ended: 3 };

export function sortEnrollments(rows: readonly BenefitEnrollmentRow[]): BenefitEnrollmentRow[] {
  return [...rows].sort(
    (a, b) =>
      (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9) ||
      b.effectiveFrom - a.effectiveFrom,
  );
}

/**
 * What the job is worth beyond pay: the compensation bar for the year, and every enrolment with
 * its tier, cost and state. An election opens beside the table; ending cover asks for a reason.
 */
export function WorkerBenefitsSection({ workerId, open, onOpen }: BenefitsViewProps) {
  const t = useT();
  const rt = useRichT();
  const queryClient = useQueryClient();
  const { closeThen } = useWorkPane();
  const { allowed: canEnroll } = usePermission(Resource.BenefitPlan, Operation.Assign);
  const [ending, setEnding] = useState<BenefitEnrollmentRow | null>(null);

  const enrollmentsQuery = useQuery({
    queryKey: [BENEFIT_ENROLLMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerBenefitEnrollments(workerId, undefined, { signal }),
  });
  const totalQuery = useQuery({
    queryKey: [TOTAL_COMPENSATION_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTotalCompensation(workerId, undefined, { signal }),
  });

  const end = useApiMutation({
    scope: { id: `benefit-end:${workerId}` },
    resourceName: "Enrollment",
    mutationFn: ({ row, reason }: { row: BenefitEnrollmentRow; reason: string }) =>
      endBenefitEnrollment({ id: row.id, notes: reason, version: row.version }),
    onSuccess: () => {
      setEnding(null);
      void queryClient.invalidateQueries({ queryKey: [BENEFIT_ENROLLMENTS_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: [TOTAL_COMPENSATION_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: ["worker"] });
    },
  });

  if (enrollmentsQuery.isPending) return <PaySkeleton view="benefits" />;
  if (enrollmentsQuery.isError) {
    return (
      <ErrorState
        error={enrollmentsQuery.error}
        layout="sheet"
        title={t("The benefits could not be loaded")}
        onRetry={() => void enrollmentsQuery.refetch()}
      />
    );
  }

  const enrollments = sortEnrollments(enrollmentsQuery.data);
  const covered = enrollments.filter((row) => row.status === "Active");
  const share = covered.reduce((sum, row) => sum + row.employeeCostMinor, 0);
  const employer = covered.reduce((sum, row) => sum + row.employerCostMinor, 0);
  const presetPlan = open.form === "enroll-benefit" ? open.kind : null;
  const presetTier = presetPlan
    ? (covered.find((row) => row.benefitPlanId === presetPlan)?.coverageTier ?? null)
    : null;
  const elect = (planId?: string) =>
    closeThen(() =>
      onOpen({
        tab: "pay",
        view: "benefits",
        form: "enroll-benefit",
        ...(planId ? { kind: planId } : {}),
      }),
    );
  const close = () => onOpen({ tab: "pay", view: "benefits" });

  return (
    <div className="flex min-w-0 flex-col gap-4 text-sm">
      {totalQuery.isPending ? (
        <div
          aria-busy
          aria-label={t("Loading the total compensation")}
          className="flex flex-col gap-3 rounded-lg border p-4"
        >
          <span className="block h-3 w-40 ui-shimmer rounded-sm" />
          <span className="block h-6 w-32 ui-shimmer rounded-sm" />
          <span className="block h-4 w-full ui-shimmer rounded-xs" />
        </div>
      ) : totalQuery.isError ? (
        <ErrorState
          error={totalQuery.error}
          layout="compact"
          title={t("The total compensation could not be loaded")}
          onRetry={() => void totalQuery.refetch()}
        />
      ) : totalQuery.data ? (
        <CompensationBar summary={totalQuery.data} />
      ) : null}

      <SectionPanel
        title={t("Enrollments")}
        count={enrollments.length}
        action={
          canEnroll && enrollments.length > 0 ? (
            <Button size="sm" variant="secondary" onClick={() => elect()}>
              <PlusIcon className="size-3.5" />
              {t("Enroll or decline")}
            </Button>
          ) : null
        }
      >
        {enrollments.length === 0 ? (
          <EmptySheet
            title={t("No enrollments recorded")}
            description={t(
              "A declined plan is worth recording too: declined and nobody asked are different facts at audit.",
            )}
            sketch={
              <div className="flex flex-col gap-2 rounded-md border bg-card p-4">
                <GhostLine className="w-1/2" />
                <GhostLine className="w-3/4" />
              </div>
            }
            action={
              canEnroll ? (
                <Button size="sm" onClick={() => elect()}>
                  {t("Enroll or decline")}
                </Button>
              ) : undefined
            }
          />
        ) : (
          <>
            <ScrollArea viewportClassName="max-h-80" maskVariant="card">
              <HoverGlideGroup>
                <table className="w-full min-w-[44rem] text-sm">
                  <thead className="sticky top-0 z-10 bg-card text-left text-xs text-muted-foreground">
                    <tr className="border-b border-border-subtle">
                      <th className="px-3 py-2 font-medium">{t("Plan")}</th>
                      <th className="px-3 py-2 font-medium">{t("Type")}</th>
                      <th className="px-3 py-2 font-medium">{t("Coverage")}</th>
                      <th className="px-3 py-2 font-medium">{t("Since")}</th>
                      <th className="px-3 py-2 text-right font-medium">{t("Cost a period")}</th>
                      <th className="px-3 py-2 font-medium">{t("Status")}</th>
                      <th className="w-0 px-2 py-2">
                        <span className="sr-only">{t("Actions")}</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {enrollments.map((row) => {
                      const active = row.status === "Active";
                      const priced = active || row.status === "Ended";
                      return (
                        <tr
                          key={row.id}
                          data-glide-target
                          data-testid={`enrollment-${row.id}`}
                          className="border-b border-border-subtle last:border-0"
                        >
                          <td className="px-3 py-2 font-medium">
                            {row.benefitPlan?.name ?? t("Plan")}
                          </td>
                          <td className="px-3 py-2">
                            {row.benefitPlan ? benefitPlanTypeLabel(row.benefitPlan.planType) : "—"}
                          </td>
                          <td className={cn("px-3 py-2", !priced && "text-muted-foreground")}>
                            {priced ? coverageTierLabel(row.coverageTier) : "—"}
                          </td>
                          <td
                            className={cn(
                              "px-3 py-2 text-xs",
                              priced ? "font-mono" : "text-muted-foreground",
                            )}
                          >
                            {priced ? formatUnixDateMedium(row.effectiveFrom) : "—"}
                          </td>
                          <td
                            className={cn(
                              "px-3 py-2 text-right text-xs",
                              active ? "font-mono" : "text-muted-foreground",
                            )}
                          >
                            {active ? formatMinor(row.employeeCostMinor) : "—"}
                          </td>
                          <td className="px-3 py-2">
                            <Badge
                              variant={enrollmentStatusTone(row.status)}
                              dot={row.status !== "Waived"}
                            >
                              {enrollmentStatusLabel(row.status)}
                            </Badge>
                            {row.status === "Waived" && row.waivedReason ? (
                              <span className="mt-0.5 block text-xs text-muted-foreground">
                                {row.waivedReason}
                              </span>
                            ) : null}
                            {row.status === "Ended" && row.effectiveTo ? (
                              <span className="mt-0.5 block text-xs text-muted-foreground tabular-nums">
                                {t("Ended {0}", formatUnixDateMedium(row.effectiveTo))}
                              </span>
                            ) : null}
                          </td>
                          <td className="px-2 py-1.5 text-right">
                            {!canEnroll ? null : active ? (
                              <RowActionsMenu
                                label={t("Enrollment options")}
                                actions={[
                                  {
                                    id: "change",
                                    label: t("Change coverage"),
                                    icon: Edit02Icon,
                                    onSelect: () => elect(row.benefitPlanId),
                                  },
                                  {
                                    id: "end",
                                    label: t("End cover…"),
                                    icon: SlashCircle01Icon,
                                    destructive: true,
                                    disabled: end.isPending,
                                    onSelect: () => setEnding(row),
                                  },
                                ]}
                              />
                            ) : row.status === "Pending" || row.status === "Waived" ? (
                              <Button
                                size="xs"
                                variant="secondary"
                                className="whitespace-nowrap"
                                onClick={() => elect(row.benefitPlanId)}
                              >
                                {t("Enroll or decline")}
                              </Button>
                            ) : null}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </HoverGlideGroup>
            </ScrollArea>
            {covered.length > 0 ? (
              <div className="flex flex-wrap justify-between gap-x-4 gap-y-1 border-t border-border-subtle py-2.5 pr-3 pl-4 text-xs text-muted-foreground tabular-nums">
                <span>
                  {rt(
                    "Their share <b>{0}</b> a period",
                    {
                      b: (chunk) => (
                        <b className="font-mono font-medium text-foreground">{chunk}</b>
                      ),
                    },
                    formatMinor(share),
                  )}
                </span>
                <span>
                  {rt(
                    "The company adds <b>{0}</b> a period",
                    {
                      b: (chunk) => (
                        <b className="font-mono font-medium text-foreground">{chunk}</b>
                      ),
                    },
                    formatMinor(employer),
                  )}
                </span>
              </div>
            ) : null}
          </>
        )}
      </SectionPanel>

      <EnrollPane
        open={open.form === "enroll-benefit" && canEnroll}
        onOpenChange={(next) => !next && close()}
        workerId={workerId}
        planId={presetPlan}
        coverageTier={presetTier}
        onSaved={close}
      />
      <ReasonDialog
        open={ending !== null}
        title={
          ending?.benefitPlan?.name
            ? t("End {0} cover?", ending.benefitPlan.name)
            : t("End this cover?")
        }
        description={t(
          "The cover ends and its deduction stops. The enrollment stays on file as ended.",
        )}
        placeholder={t("e.g. Moving to a spouse's plan")}
        confirmLabel={t("End cover")}
        destructive
        maxLength={2000}
        pending={end.isPending}
        onConfirm={(reason) => {
          if (ending) void end.mutateAsync({ row: ending, reason }).catch(() => undefined);
        }}
        onOpenChange={(next) => !next && !end.isPending && setEnding(null)}
      />
    </div>
  );
}
