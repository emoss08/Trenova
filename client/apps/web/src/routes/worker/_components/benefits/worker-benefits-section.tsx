import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  BENEFIT_ENROLLMENTS_KEY,
  endBenefitEnrollment,
  fetchWorkerBenefitEnrollments,
  fetchWorkerTotalCompensation,
  TOTAL_COMPENSATION_KEY,
} from "@/lib/graphql/benefits";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  benefitPlanTypeLabel,
  coverageTierLabel,
  employerSharePercent,
  enrollmentStatusLabel,
  enrollmentStatusTone,
  formatMinor,
} from "@trenova/shared/lib/benefits";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { EnrollDialog } from "./enroll-dialog";

/**
 * A worker's cover and what the package is worth. The employee's own
 * contributions are shown but never added into the total: money the worker paid
 * is not money the job gave them.
 */
export function WorkerBenefitsSection({ workerId }: { workerId: string }) {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.BenefitPlan, Operation.Read);
  const { allowed: canEnroll } = usePermission(Resource.BenefitPlan, Operation.Assign);
  const [dialogOpen, setDialogOpen] = useState(false);

  const enrollmentsQuery = useQuery({
    queryKey: [BENEFIT_ENROLLMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerBenefitEnrollments(workerId, undefined, { signal }),
    enabled: canRead,
  });
  const totalQuery = useQuery({
    queryKey: [TOTAL_COMPENSATION_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTotalCompensation(workerId, undefined, { signal }),
    enabled: canRead,
  });

  const endMutation = useMutation({
    mutationFn: (id: string) => endBenefitEnrollment({ id }),
    onSuccess: () => {
      toast.success(t("Cover ended"), {
        description: t("The contribution stops with it; the record of the cover stays."),
      });
      void queryClient.invalidateQueries({ queryKey: [BENEFIT_ENROLLMENTS_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: [TOTAL_COMPENSATION_KEY, workerId] });
    },
    onError: (error: Error) =>
      toast.error(t("Could not end the cover"), { description: error.message }),
  });

  if (!canRead) return null;

  if (enrollmentsQuery.isLoading) return <Skeleton className="h-24 w-full" />;

  const enrollments = enrollmentsQuery.data ?? [];
  const total = totalQuery.data;

  return (
    <section>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h3 className="cc-label text-foreground">{t("Benefits")}</h3>
        {canEnroll ? (
          <Button size="sm" variant="outline" onClick={() => setDialogOpen(true)}>
            <PlusIcon className="size-3.5" />
            {t("Enroll or decline")}
          </Button>
        ) : null}
      </div>

      {total ? (
        <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Figure label={t("Paid this year")} value={formatMinor(total.grossPayMinor)} />
          <Figure
            label={t("Employer benefits")}
            value={formatMinor(total.employerBenefitMinor)}
            detail={t("Per settlement period")}
          />
          <Figure
            label={t("Their contribution")}
            value={formatMinor(total.employeeBenefitMinor)}
            detail={t("Not part of the total")}
          />
          <Figure
            label={t("Total compensation")}
            value={formatMinor(total.totalCompensationMinor)}
            detail={`${employerSharePercent(
              total.employerBenefitMinor,
              total.totalCompensationMinor,
            )}% is benefits`}
          />
        </div>
      ) : null}

      {enrollments.length === 0 ? (
        <p className="text-muted-foreground rounded-md border border-dashed p-3 text-xs">
          {t("Nothing recorded. A declined plan is worth recording too — “declined” and “nobody asked” are different facts at audit.")}
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {enrollments.map((enrollment) => (
            <li
              key={enrollment.id}
              className="flex flex-wrap items-center justify-between gap-2 border-t pt-1.5 text-xs first:border-t-0 first:pt-0"
            >
              <span className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="font-medium">{enrollment.benefitPlan?.name ?? "Plan"}</span>
                {enrollment.benefitPlan ? (
                  <Badge variant="secondary">
                    {benefitPlanTypeLabel(enrollment.benefitPlan.planType)}
                  </Badge>
                ) : null}
                <Badge variant={enrollmentStatusTone(enrollment.status)}>
                  {enrollmentStatusLabel(enrollment.status)}
                </Badge>
                <span className="text-muted-foreground">
                  {coverageTierLabel(enrollment.coverageTier)}
                </span>
                <span className="text-muted-foreground">
                  {t("from {0} {1}", formatUnixDate(enrollment.effectiveFrom), enrollment.effectiveTo ? ` to ${formatUnixDate(enrollment.effectiveTo)}` : "")}
                </span>
                {enrollment.waivedReason ? (
                  <span className="text-muted-foreground truncate">{enrollment.waivedReason}</span>
                ) : null}
              </span>
              <span className="flex shrink-0 items-center gap-2 tabular-nums">
                {enrollment.status === "Active" ? (
                  <span className="text-muted-foreground">
                    {t("{0} /period", formatMinor(enrollment.employeeCostMinor))}
                  </span>
                ) : null}
                {canEnroll && enrollment.status !== "Ended" ? (
                  <Button
                    size="xs"
                    variant="ghost"
                    isLoading={endMutation.isPending}
                    onClick={() => endMutation.mutate(enrollment.id)}
                    aria-label={`End ${enrollment.benefitPlan?.name ?? "cover"}`}
                  >
                    {t("End cover")}
                  </Button>
                ) : null}
              </span>
            </li>
          ))}
        </ul>
      )}

      <EnrollDialog open={dialogOpen} onOpenChange={setDialogOpen} workerId={workerId} />
    </section>
  );
}

function Figure({ label, value, detail }: { label: string; value: string; detail?: string }) {
  return (
    <div className="rounded-md border px-2.5 py-1.5">
      <p className="text-muted-foreground text-[11px]">{label}</p>
      <p className="text-sm font-semibold tabular-nums">{value}</p>
      {detail ? <p className="text-muted-foreground text-[11px]">{detail}</p> : null}
    </div>
  );
}
