import { usePermission } from "@/hooks/use-permission";
import { type BenefitPlanRow } from "@/lib/graphql/benefits";
import { costTotals, defaultPlanYear, emptyActivePlans, planYearsOf } from "@/lib/benefits-console";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { BenefitPlanDialog } from "./benefit-plan-dialog";
import { BenefitsEmpty } from "./benefits-empty";
import { BenefitsAside } from "./benefits-aside";
import { BenefitsOverview } from "./benefits-overview";
import { BenefitsSkeleton } from "./benefits-skeleton";
import { PlanEnrollmentsSheet } from "./plan-enrollments-sheet";
import { PlanGroups } from "./plan-groups";
import {
  benefitCostsQuery,
  benefitPlansQuery,
  declinedEnrollmentsQuery,
  openEnrollmentsQuery,
} from "./queries";

const EMPTY_PLANS: BenefitPlanRow[] = [];

export default function BenefitsConsole() {
  const { allowed: canRead } = usePermission(Resource.BenefitPlan, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.BenefitPlan, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.BenefitPlan, Operation.Update);
  const [dialog, setDialog] = useState<{ plan: BenefitPlanRow | null } | null>(null);
  const [roster, setRoster] = useState<BenefitPlanRow | null>(null);
  const [chosenYear, setChosenYear] = useState<number | null>(null);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const thisYear = new Date(now * 1000).getUTCFullYear();

  const plansQuery = useQuery({ ...benefitPlansQuery(), enabled: canRead });
  const plans = plansQuery.data ?? EMPTY_PLANS;
  const years = useMemo(() => planYearsOf(plans), [plans]);
  // The chosen year survives only while a plan exists for it: archiving the
  // last plan of a year must not leave the page reading an empty year.
  const planYear =
    chosenYear !== null && years.includes(chosenYear)
      ? chosenYear
      : defaultPlanYear(years, thisYear);

  const costsQuery = useQuery({
    ...benefitCostsQuery(planYear),
    enabled: canRead && planYear !== null,
  });
  const openQuery = useQuery({ ...openEnrollmentsQuery(), enabled: canRead });
  const declinedQuery = useQuery({ ...declinedEnrollmentsQuery(), enabled: canRead });

  const yearPlans = useMemo(
    () => plans.filter((plan) => plan.planYear === planYear),
    [plans, planYear],
  );
  const costs = costsQuery.data;
  const totals = useMemo(() => costTotals(costs ?? []), [costs]);
  const empty = useMemo(() => emptyActivePlans(yearPlans, costs ?? []), [yearPlans, costs]);
  const yearItems = useMemo(
    () => years.map((year) => ({ value: String(year), label: String(year) })),
    [years],
  );

  if (!canRead) return null;

  if (plansQuery.isLoading) return <BenefitsSkeleton />;

  if (plans.length === 0) {
    return (
      <div className="flex flex-col">
        <BenefitsEmpty
          title="No plans yet"
          description={
            "A plan needs a pay code so a contribution can be categorised on a settlement. " +
            "Add one, then put workers on it from their Benefits tab."
          }
          onAddPlan={canCreate ? () => setDialog({ plan: null }) : undefined}
        />
        <BenefitPlanDialog
          open={dialog !== null}
          onOpenChange={(open) => !open && setDialog(null)}
          plan={dialog?.plan ?? null}
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <BenefitsOverview
        plans={plans}
        costs={costs}
        openEnrollments={openQuery.data}
        planYear={planYear}
        now={now}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3">
          {yearItems.length > 1 ? (
            <SegmentedControl<string>
              items={yearItems}
              value={String(planYear)}
              onValueChange={(value) => setChosenYear(Number(value))}
              aria-label="Plan year"
            />
          ) : (
            <span className="text-sm font-medium tabular-nums">Plan year {planYear}</span>
          )}
          <p className="text-muted-foreground text-xs">
            Each plan year is its own row, so repricing next year never restates what somebody was
            charged this year.
            {empty.length > 0
              ? ` ${empty.length} active plan${empty.length === 1 ? " has" : "s have"} nobody on ${empty.length === 1 ? "it" : "them"}.`
              : ""}
          </p>
        </div>
        {canCreate ? (
          <Button size="sm" onClick={() => setDialog({ plan: null })}>
            <PlusIcon className="size-3.5" />
            Add a plan
          </Button>
        ) : null}
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="flex min-w-0 flex-col gap-4">
          {yearPlans.length === 0 ? (
            <BenefitsEmpty
              title={`No plans for ${planYear}`}
              description="Every plan year is priced on its own, so a year with nothing on offer stays empty until a plan is added for it."
              onAddPlan={canCreate ? () => setDialog({ plan: null }) : undefined}
            />
          ) : (
            <PlanGroups
              plans={yearPlans}
              costs={costs ?? []}
              totalEnrolled={totals.enrolled}
              canUpdate={canUpdate}
              onEdit={(plan) => setDialog({ plan })}
              onOpenRoster={setRoster}
            />
          )}
        </div>
        <BenefitsAside openEnrollments={openQuery.data} declined={declinedQuery.data} now={now} />
      </div>

      <BenefitPlanDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        plan={dialog?.plan ?? null}
      />
      <PlanEnrollmentsSheet
        plan={roster}
        now={now}
        onOpenChange={(open) => !open && setRoster(null)}
      />
    </div>
  );
}
