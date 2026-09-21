import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader } from "@/components/kpi/kpi-card";
import { KPI_VALUE_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type {
  BenefitCostRow,
  BenefitEnrollmentListRow,
  BenefitPlanRow,
} from "@/lib/graphql/benefits";
import { costTotals, endingSoon, startingSoon } from "@/lib/benefits-console";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { useMemo } from "react";

type BenefitsOverviewProps = {
  plans: readonly BenefitPlanRow[] | undefined;
  costs: readonly BenefitCostRow[] | undefined;
  openEnrollments: readonly BenefitEnrollmentListRow[] | undefined;
  planYear: number | null;
  now: number;
};

/**
 * The year in four numbers: who is covered, what is on offer, what the
 * employer puts in and what comes off settlements. Read from the same cost
 * rows the plan list shows, so the headline never disagrees with the detail.
 */
export function BenefitsOverview({
  plans,
  costs,
  openEnrollments,
  planYear,
  now,
}: BenefitsOverviewProps) {
  const t = useT();

  const totals = useMemo(() => costTotals(costs ?? []), [costs]);
  const yearPlans = useMemo(
    () => (plans ?? []).filter((plan) => planYear === null || plan.planYear === planYear),
    [plans, planYear],
  );
  const activePlans = yearPlans.filter((plan) => plan.status === "Active").length;
  const archivedPlans = yearPlans.length - activePlans;
  const starting = useMemo(
    () => startingSoon(openEnrollments ?? [], now).length,
    [openEnrollments, now],
  );
  const ending = useMemo(
    () => endingSoon(openEnrollments ?? [], now).length,
    [openEnrollments, now],
  );

  return (
    <KpiStrip>
      <KpiStripItem
        label={t("People covered")}
        info={
          <InfoPopover title={t("People covered")}>
            {t(
              "Active enrolments in the selected plan year, summed across plans. Somebody on two plans counts twice.",
            )}
          </InfoPopover>
        }
        value={
          costs ? (
            <NumberFlow value={totals.enrolled} aria-label={t("People covered")} />
          ) : (
            <Skeleton className="h-6 w-10" />
          )
        }
        sub={describeCovered(totals.waived, starting, ending)}
      />

      <KpiStripItem
        label={t("Plans on offer")}
        info={
          <InfoPopover title={t("Plans on offer")}>
            {t(
              "Plans in the year still open to enrolment. Archived plans keep their enrolments but are not counted.",
            )}
          </InfoPopover>
        }
        value={
          plans ? (
            <NumberFlow value={activePlans} aria-label={t("Plans on offer")} />
          ) : (
            <Skeleton className="h-6 w-10" />
          )
        }
        sub={
          planYear === null
            ? t("Nothing on file yet")
            : archivedPlans > 0
              ? t("For {0} · {1} archived", planYear, archivedPlans)
              : t("For {0}", planYear)
        }
      />

      <KpiCard span={2}>
        <KpiHeader
          label={t("Employer puts in")}
          info={
            <InfoPopover title={t("Employer puts in")}>
              {t(
                "The employer share across every active enrolment in the year, at each plan's stated rates.",
              )}
            </InfoPopover>
          }
        />
        {costs ? (
          <span className={KPI_VALUE_CLASS} aria-label={t("Employer puts in")}>
            {formatMinor(totals.employerMinor)}
          </span>
        ) : (
          <Skeleton className="h-6 w-24" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("Who pays, per settlement period")}
          formatValue={(value) => formatMinor(value)}
          segments={[
            { key: "employer", label: t("Employer"), value: totals.employerMinor },
            { key: "employee", label: t("Employees"), value: totals.employeeMinor },
          ]}
        />
      </KpiCard>

      <KpiStripItem
        label={t("Off settlements")}
        info={
          <InfoPopover title={t("Off settlements")}>
            {t("The employee share across the same enrolments, deducted from driver settlements.")}
          </InfoPopover>
        }
        value={
          costs ? (
            <span aria-label={t("Off settlements")}>{formatMinor(totals.employeeMinor)}</span>
          ) : (
            <Skeleton className="h-6 w-24" />
          )
        }
        sub={t("Per settlement period, taken as ordinary deductions")}
      />
    </KpiStrip>
  );
}

function describeCovered(waived: number, starting: number, ending: number): string {
  const parts: string[] = [];
  if (waived > 0) parts.push(`${waived} declined`);
  if (starting > 0) parts.push(`${starting} starting soon`);
  if (ending > 0) parts.push(`${ending} ending soon`);
  return parts.length > 0 ? parts.join(" · ") : "Nobody declined, nothing in flux";
}
