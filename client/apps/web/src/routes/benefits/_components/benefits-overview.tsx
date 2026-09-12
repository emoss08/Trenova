import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
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
import { BanknoteIcon, HeartHandshakeIcon, LayersIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

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
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label={t("People covered")}
          info={
            <InfoPopover title={t("People covered")}>
              {
                t("Active enrolments in the selected plan year, summed across plans. Somebody on two plans counts twice.")
              }
            </InfoPopover>
          }
        />
        {costs ? (
          <NumberFlow value={totals.enrolled} className={VALUE_CLASS} aria-label={t("People covered")} />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>{describeCovered(totals.waived, starting, ending)}</KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<LayersIcon className="size-[11px]" />}
          label={t("Plans on offer")}
          info={
            <InfoPopover title={t("Plans on offer")}>
              {
                t("Plans in the year still open to enrolment. Archived plans keep their enrolments but are not counted.")
              }
            </InfoPopover>
          }
        />
        {plans ? (
          <NumberFlow value={activePlans} className={VALUE_CLASS} aria-label={t("Plans on offer")} />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {planYear === null
            ? t("Nothing on file yet")
            : archivedPlans > 0
              ? t("For {0} · {1} archived", planYear, archivedPlans)
              : t("For {0}", planYear)}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<HeartHandshakeIcon className="size-[11px]" />}
          label={t("Employer puts in")}
          info={
            <InfoPopover title={t("Employer puts in")}>
              {
                t("The employer share across every active enrolment in the year, at each plan's stated rates.")
              }
            </InfoPopover>
          }
        />
        {costs ? (
          <span className={VALUE_CLASS} aria-label={t("Employer puts in")}>
            {formatMinor(totals.employerMinor)}
          </span>
        ) : (
          <Skeleton className="h-6.5 w-24" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("Who pays, per settlement period")}
          formatValue={(value) => formatMinor(value)}
          segments={[
            { key: "employer", label: "Employer", value: totals.employerMinor },
            { key: "employee", label: "Employees", value: totals.employeeMinor },
          ]}
        />
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<BanknoteIcon className="size-[11px]" />}
          label={t("Off settlements")}
          info={
            <InfoPopover title={t("Off settlements")}>
              {t("The employee share across the same enrolments, deducted from driver settlements.")}
            </InfoPopover>
          }
        />
        {costs ? (
          <span className={VALUE_CLASS} aria-label={t("Off settlements")}>
            {formatMinor(totals.employeeMinor)}
          </span>
        ) : (
          <Skeleton className="h-6.5 w-24" />
        )}
        <KpiSub>{t("Per settlement period, taken as ordinary deductions")}</KpiSub>
      </KpiCard>
    </div>
  );
}

function describeCovered(waived: number, starting: number, ending: number): string {
  const parts: string[] = [];
  if (waived > 0) parts.push(`${waived} declined`);
  if (starting > 0) parts.push(`${starting} starting soon`);
  if (ending > 0) parts.push(`${ending} ending soon`);
  return parts.length > 0 ? parts.join(" · ") : "Nobody declined, nothing in flux";
}
