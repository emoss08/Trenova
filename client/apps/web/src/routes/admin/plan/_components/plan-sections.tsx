import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { ManagedPlan, ManagedPlanAddOn, ManagedPlanChange } from "@/lib/graphql/managed-plan";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";

function featureNames(addOn: Pick<ManagedPlanAddOn, "features">): string {
  return formatList(addOn.features.map((feature) => feature.name));
}

function AddOnRow({ addOn, onPlan }: { addOn: ManagedPlanAddOn; onPlan: boolean }) {
  const t = useT();

  return (
    <li className="flex flex-col gap-1 px-3 py-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-medium text-foreground">{addOn.name}</span>
        {onPlan ? (
          <Badge variant="success" dot>
            {t("On your plan")}
          </Badge>
        ) : null}
      </div>
      <p className="text-xs text-muted-foreground">
        {onPlan
          ? t("Adds {0}.", featureNames(addOn))
          : t("Would add {0}. Add it from your Trenova account.", featureNames(addOn))}
      </p>
    </li>
  );
}

function ChangeRow({ change }: { change: ManagedPlanChange }) {
  const t = useT();
  const joined = change.joined.map((addOn) => addOn.name);
  const left = change.left.map((addOn) => addOn.name);

  return (
    <li className="flex flex-col gap-1 px-3 py-3">
      <span className="text-xs text-muted-foreground tabular-nums">
        {formatUnixDate(change.changedAt)}
      </span>
      {change.starting ? (
        <p className="text-sm text-foreground">
          {joined.length > 0
            ? t("Your plan started with {0}.", formatList(joined))
            : t("Your plan started with the base plan alone.")}
        </p>
      ) : null}
      {!change.starting && joined.length > 0 ? (
        <p className="text-sm text-foreground">{t("Joined your plan: {0}.", formatList(joined))}</p>
      ) : null}
      {left.length > 0 ? (
        <p className="text-sm text-foreground">
          {t("Left your plan: {0}.", formatList(left))}{" "}
          {t(
            "{0, plural, one {Everything you entered with it is kept and comes back if you add it again.} other {Everything you entered with them is kept and comes back if you add them again.}}",
            left.length,
          )}
        </p>
      ) : null}
    </li>
  );
}

/** What the plan includes, the add-ons on it and off it, and the changes that moved one. */
export function PlanSections({ plan }: { plan: ManagedPlan }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel title={t("Included in your plan")}>
        <p className="px-3 py-3 text-sm text-foreground">
          {formatList(plan.base.map((feature) => feature.name))}
        </p>
      </SectionPanel>

      <SectionPanel title={t("Add-ons on your plan")} count={plan.included.length}>
        {plan.included.length > 0 ? (
          <ul className="flex flex-col divide-y divide-border">
            {plan.included.map((addOn) => (
              <AddOnRow key={addOn.key} addOn={addOn} onPlan />
            ))}
          </ul>
        ) : (
          <SectionPanelQuiet>
            {t("Your plan has no add-ons. It includes everything listed above.")}
          </SectionPanelQuiet>
        )}
      </SectionPanel>

      <SectionPanel title={t("Add-ons you can add")} count={plan.available.length}>
        {plan.available.length > 0 ? (
          <ul className="flex flex-col divide-y divide-border">
            {plan.available.map((addOn) => (
              <AddOnRow key={addOn.key} addOn={addOn} onPlan={false} />
            ))}
          </ul>
        ) : (
          <SectionPanelQuiet>{t("Every add-on is on your plan.")}</SectionPanelQuiet>
        )}
      </SectionPanel>

      <SectionPanel title={t("Recent changes")}>
        {plan.changes.length > 0 ? (
          <ul className="flex flex-col divide-y divide-border">
            {plan.changes.map((change, index) => (
              <ChangeRow key={`${change.changedAt}:${index}`} change={change} />
            ))}
          </ul>
        ) : (
          <SectionPanelQuiet>{t("Your plan has not changed yet.")}</SectionPanelQuiet>
        )}
      </SectionPanel>
    </div>
  );
}
