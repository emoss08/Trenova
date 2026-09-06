import { usePermission } from "@/hooks/use-permission";
import {
  BENEFIT_COSTS_KEY,
  BENEFIT_PLANS_KEY,
  fetchBenefitCosts,
  fetchBenefitPlans,
  type BenefitPlanRow,
} from "@/lib/graphql/benefits";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { benefitPlanTypeLabel, formatMinor } from "@trenova/shared/lib/benefits";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { BenefitPlanDialog } from "./benefit-plan-dialog";

export default function BenefitsConsole() {
  const { allowed: canRead } = usePermission(Resource.BenefitPlan, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.BenefitPlan, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.BenefitPlan, Operation.Update);
  const [dialog, setDialog] = useState<{ plan: BenefitPlanRow | null } | null>(null);

  const plansQuery = useQuery({
    queryKey: [BENEFIT_PLANS_KEY],
    queryFn: ({ signal }) => fetchBenefitPlans(undefined, { signal }),
    enabled: canRead,
  });
  const costsQuery = useQuery({
    queryKey: [BENEFIT_COSTS_KEY],
    queryFn: ({ signal }) => fetchBenefitCosts(undefined, { signal }),
    enabled: canRead,
  });

  const totals = useMemo(() => {
    const rows = costsQuery.data ?? [];
    return rows.reduce(
      (acc, row) => ({
        enrolled: acc.enrolled + row.enrolled,
        waived: acc.waived + row.waived,
        employee: acc.employee + row.employeeCostMinor,
        employer: acc.employer + row.employerCostMinor,
      }),
      { enrolled: 0, waived: 0, employee: 0, employer: 0 },
    );
  }, [costsQuery.data]);

  if (!canRead) return null;

  if (plansQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const plans = plansQuery.data ?? [];
  const costs = costsQuery.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <section className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Figure label="People covered" value={String(totals.enrolled)} />
        <Figure label="Declined" value={String(totals.waived)} detail="Recorded, not missing" />
        <Figure
          label="Employer cost"
          value={formatMinor(totals.employer)}
          detail="Per settlement period"
        />
        <Figure
          label="Employee cost"
          value={formatMinor(totals.employee)}
          detail="Taken as deductions"
        />
      </section>

      <section className="rounded-lg border p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-medium">Plans</h3>
            <p className="text-muted-foreground text-xs">
              Each plan year is its own row, so repricing next year never restates what somebody was
              charged this year.
            </p>
          </div>
          {canCreate ? (
            <Button size="sm" onClick={() => setDialog({ plan: null })}>
              <PlusIcon className="size-3.5" />
              Add a plan
            </Button>
          ) : null}
        </div>

        {plans.length === 0 ? (
          <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
            No plans yet. A plan needs a pay code so a contribution can be categorised on a
            settlement.
          </p>
        ) : (
          <ul className="mt-3 flex flex-col gap-1.5">
            {plans.map((plan) => {
              const cost = costs.find((row) => row.planId === plan.id);
              return (
                <li
                  key={plan.id}
                  className="flex flex-wrap items-center justify-between gap-2 border-t pt-1.5 text-xs first:border-t-0 first:pt-0"
                >
                  <span className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="font-medium">{plan.name}</span>
                    <span className="text-muted-foreground tabular-nums">{plan.code}</span>
                    <Badge variant="secondary">{benefitPlanTypeLabel(plan.planType)}</Badge>
                    <Badge variant="outline">{plan.planYear}</Badge>
                    {plan.status !== "Active" ? <Badge variant="inactive">Archived</Badge> : null}
                    {plan.carrier ? (
                      <span className="text-muted-foreground truncate">{plan.carrier}</span>
                    ) : null}
                  </span>
                  <span className="flex shrink-0 items-center gap-3 tabular-nums">
                    <span className="text-muted-foreground">
                      {formatMinor(plan.employeeCostMinor, plan.currencyCode)} employee ·{" "}
                      {formatMinor(plan.employerCostMinor, plan.currencyCode)} employer
                    </span>
                    {cost ? (
                      <Badge variant={cost.enrolled > 0 ? "active" : "secondary"}>
                        {cost.enrolled} on it
                      </Badge>
                    ) : null}
                    {canUpdate ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        onClick={() => setDialog({ plan })}
                        aria-label={`Edit ${plan.name}`}
                      >
                        Edit
                      </Button>
                    ) : null}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <BenefitPlanDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        plan={dialog?.plan ?? null}
      />
    </div>
  );
}

function Figure({ label, value, detail }: { label: string; value: string; detail?: string }) {
  return (
    <div className="rounded-lg border px-3 py-2">
      <p className="text-muted-foreground text-[11px]">{label}</p>
      <p className="text-lg font-semibold tabular-nums">{value}</p>
      {detail ? <p className="text-muted-foreground text-[11px]">{detail}</p> : null}
    </div>
  );
}
