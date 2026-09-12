import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  benefitPlanTypeLabel,
  coverageTierLabel,
  employerSharePercent,
  formatMinor,
} from "@trenova/shared/lib/benefits";
import { fetchMyTotalCompensation } from "@trenova/shared/lib/graphql/driver-portal";
import { useQuery } from "@tanstack/react-query";

/**
 * What the job is actually worth, not just what landed in the bank. The
 * employer's benefit contributions are added; the driver's own are shown
 * separately and never added, because money they paid is not money the job
 * gave them.
 */
export function TotalCompCard() {
  const t = useT();

  const total = useQuery({
    queryKey: ["dash-total-comp"],
    queryFn: ({ signal }) => fetchMyTotalCompensation({ signal }),
    staleTime: 5 * 60 * 1000,
  });

  if (total.isPending) {
    return <Skeleton className="h-36 w-full rounded-2xl" />;
  }
  if (!total.data) return null;

  const data = total.data;
  // Nothing paid and no cover means there is nothing to state yet.
  if (data.totalCompensationMinor <= 0 && data.enrollments.length === 0) return null;

  const share = employerSharePercent(data.employerBenefitMinor, data.totalCompensationMinor);

  return (
    <div className="border-border bg-card rounded-2xl border p-4" data-testid="total-comp-card">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold">{t("Total compensation")}</p>
          <p className="text-muted-foreground text-xs">{t("Plan year {0}", data.planYear)}</p>
        </div>
      </div>

      <p className="mt-3 text-2xl font-semibold tabular-nums">
        {formatMinor(data.totalCompensationMinor)}
      </p>
      <p className="text-muted-foreground text-xs">
        {t(
          "{0} paid {1} {2}",
          formatMinor(data.grossPayMinor),
          data.employerBenefitMinor > 0
            ? ` ${t("· {0} the company puts in", formatMinor(data.employerBenefitMinor))}`
            : "",
          share > 0 ? ` ${t("· {0}% of it is benefits", share)}` : "",
        )}
      </p>

      {data.employeeBenefitMinor > 0 ? (
        <p className="text-muted-foreground mt-2 text-xs">
          {t(
            "You contribute {0} a period. That is not counted in the figure above — it is money you paid, not money the job gave you.",
            formatMinor(data.employeeBenefitMinor),
          )}
        </p>
      ) : null}

      {data.enrollments.length > 0 ? (
        <ul className="border-border mt-3 flex flex-col gap-1.5 border-t pt-3">
          {data.enrollments.map((enrollment) => (
            <li key={enrollment.id} className="flex items-center justify-between gap-2 text-xs">
              <span className="flex min-w-0 items-center gap-1.5">
                <span className="truncate font-medium">
                  {enrollment.benefitPlan?.name ?? t("Cover")}
                </span>
                {enrollment.benefitPlan ? (
                  <Badge variant="secondary">
                    {benefitPlanTypeLabel(enrollment.benefitPlan.planType)}
                  </Badge>
                ) : null}
                <span className="text-muted-foreground truncate">
                  {coverageTierLabel(enrollment.coverageTier)}
                </span>
              </span>
              <span className="text-muted-foreground shrink-0 tabular-nums">
                {t(
                  "{0} you · {1} them",
                  formatMinor(enrollment.employeeCostMinor),
                  formatMinor(enrollment.employerCostMinor),
                )}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
