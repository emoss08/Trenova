import { useT } from "@trenova/shared/i18n/use-t";
import type { BenefitCostRow, BenefitPlanRow } from "@/lib/graphql/benefits";
import { groupPlansByType, type PlanTypeGroup } from "@/lib/benefits-console";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { headcountShare } from "@trenova/shared/lib/org-structure";
import { cn } from "@trenova/shared/lib/utils";
import { PencilLineIcon, UsersIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";

const STAGGER_LIMIT = 10;

type PlanGroupsProps = {
  plans: readonly BenefitPlanRow[];
  costs: readonly BenefitCostRow[];
  /** Everyone covered across the year, the denominator for each plan's share. */
  totalEnrolled: number;
  canUpdate: boolean;
  onEdit: (plan: BenefitPlanRow) => void;
  onOpenRoster: (plan: BenefitPlanRow) => void;
};

/**
 * Plans by kind, each with what it costs and who is on it. The kind heading
 * carries the kind's own totals, so "what does medical cost us" is answered
 * without adding rows up by eye.
 */
export function PlanGroups({
  plans,
  costs,
  totalEnrolled,
  canUpdate,
  onEdit,
  onOpenRoster,
}: PlanGroupsProps) {
  const groups = useMemo(() => groupPlansByType(plans, costs), [plans, costs]);
  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  let index = 0;
  return (
    <div className="flex flex-col gap-4">
      {groups.map((group) => (
        <section key={group.type} aria-label={group.label} className="flex flex-col gap-1.5">
          <GroupHeading group={group} />
          <ul className="bg-card divide-y overflow-hidden rounded-lg border">
            {group.plans.map(({ plan, cost }) => {
              const position = index++;
              return (
                <m.li
                  key={plan.id}
                  initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.25, delay: Math.min(position, STAGGER_LIMIT) * 0.03 }}
                >
                  <PlanRow
                    plan={plan}
                    cost={cost}
                    totalEnrolled={totalEnrolled}
                    canUpdate={canUpdate}
                    onEdit={() => onEdit(plan)}
                    onOpenRoster={() => onOpenRoster(plan)}
                  />
                </m.li>
              );
            })}
          </ul>
        </section>
      ))}
    </div>
  );
}

function GroupHeading({ group }: { group: PlanTypeGroup<BenefitPlanRow, BenefitCostRow> }) {
  const t = useT();

  return (
    <header className="flex flex-wrap items-baseline justify-between gap-2 px-1">
      <h4 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
        {group.label}
        <span className="ml-1.5 font-normal normal-case tabular-nums">
          {t("{0} plan{1}", group.plans.length, group.plans.length === 1 ? "" : "s")}
        </span>
      </h4>
      <span className="text-muted-foreground text-xs tabular-nums">
        {t("{0} covered {1}{2}", group.enrolled, group.waived > 0 ? ` · ${group.waived} declined` : "", group.employerMinor > 0 ? ` · ${formatMinor(group.employerMinor)} employer` : "")}
      </span>
    </header>
  );
}

type PlanRowProps = {
  plan: BenefitPlanRow;
  cost: BenefitCostRow | null;
  totalEnrolled: number;
  canUpdate: boolean;
  onEdit: () => void;
  onOpenRoster: () => void;
};

function PlanRow({ plan, cost, totalEnrolled, canUpdate, onEdit, onOpenRoster }: PlanRowProps) {
  const t = useT();

  const archived = plan.status !== "Active";
  const enrolled = cost?.enrolled ?? 0;
  const waived = cost?.waived ?? 0;
  const detail = [
    plan.carrier,
    plan.policyNumber ? `Policy ${plan.policyNumber}` : null,
    plan.payCode ? `Pay code ${plan.payCode.code}` : null,
    plan.waitingPeriodDays > 0 ? `${plan.waitingPeriodDays}-day wait` : "No waiting period",
  ].filter(Boolean);

  return (
    <article
      aria-label={plan.name}
      className={cn(
        "grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5",
        archived && "opacity-70",
      )}
    >
      <div className="flex min-w-0 flex-col gap-1">
        <span className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="truncate text-sm font-medium">{plan.name}</span>
          <span className="text-muted-foreground text-xs tabular-nums">{plan.code}</span>
          {archived ? <Badge variant="inactive">{t("Archived")}</Badge> : null}
        </span>
        <span className="text-muted-foreground truncate text-xs">{detail.join(" · ")}</span>
        <div className="flex items-center gap-2">
          <span
            role="img"
            aria-label={`${plan.name}: ${enrolled} of ${totalEnrolled} covered people`}
            className="bg-muted flex h-1 w-32 overflow-hidden rounded-full"
          >
            <span
              aria-hidden
              className="bg-brand/60 h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none"
              style={{ width: `${headcountShare(enrolled, totalEnrolled)}%` }}
            />
          </span>
          <span className="text-muted-foreground text-2xs tabular-nums">
            {t("{0} on it{1}", enrolled, waived > 0 ? ` · ${waived} declined` : "")}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <dl className="hidden text-right text-xs tabular-nums sm:grid sm:grid-cols-[auto_auto] sm:gap-x-2 sm:gap-y-0.5">
          <dt className="text-muted-foreground">{t("Employee")}</dt>
          <dd className="font-mono font-medium">
            {formatMinor(plan.employeeCostMinor, plan.currencyCode)}
          </dd>
          <dt className="text-muted-foreground">{t("Employer")}</dt>
          <dd className="font-mono font-medium">
            {formatMinor(plan.employerCostMinor, plan.currencyCode)}
          </dd>
        </dl>
        <div className="flex items-center gap-1">
          <Button size="xs" variant="outline" onClick={onOpenRoster}>
            <UsersIcon className="size-3" />
            {t("Who is on it")}
          </Button>
          {canUpdate ? (
            <Button
              size="icon-xs"
              variant="ghost"
              onClick={onEdit}
              aria-label={`Edit ${plan.name}`}
            >
              <PencilLineIcon className="size-3.5" />
            </Button>
          ) : null}
        </div>
      </div>
    </article>
  );
}
