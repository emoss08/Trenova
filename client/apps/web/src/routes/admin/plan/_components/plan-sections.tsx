import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { ManagedPlan, ManagedPlanAddOn, ManagedPlanChange } from "@/lib/graphql/managed-plan";
import { ArrowUpRightIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixMonthDay } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { Fragment, useId, useState } from "react";
import { addOnUrl } from "./add-on-url";
import { PlanEndNotice } from "./plan-end-notice";
import { PlanUsage } from "./plan-usage";

function AddOnState({ addOn }: { addOn: ManagedPlanAddOn }) {
  const t = useT();

  if (addOn.endsAt != null) {
    return (
      <Badge variant="warning" dot>
        {t("Ends {0}", formatUnixMonthDay(addOn.endsAt))}
      </Badge>
    );
  }
  return (
    <Badge variant="success" dot>
      {t("On")}
    </Badge>
  );
}

function AddLink({ addOn, href }: { addOn: ManagedPlanAddOn; href: string }) {
  const t = useT();

  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={t("Add {0} from your Trenova account", addOn.name)}
      className={cn(buttonVariants({ variant: "ghost", size: "xs" }), "text-brand")}
    >
      {t("Add")}
      <ArrowUpRightIcon className="size-3.5" aria-hidden />
    </a>
  );
}

function AddOnRow({
  addOn,
  onPlan,
  manageUrl,
}: {
  addOn: ManagedPlanAddOn;
  onPlan: boolean;
  manageUrl: string | null;
}) {
  const [open, setOpen] = useState(false);
  const featuresId = useId();
  const addHref = !onPlan && manageUrl ? addOnUrl(manageUrl, addOn.key) : null;
  const side = onPlan ? (
    <AddOnState addOn={addOn} />
  ) : addHref ? (
    <AddLink addOn={addOn} href={addHref} />
  ) : null;

  return (
    <li className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-0.5 border-t border-border-subtle px-3 py-2.5 max-sm:first:border-t-0 sm:odd:border-r sm:nth-[-n+2]:border-t-0">
      <Button
        variant="bare"
        size="bare"
        aria-expanded={open}
        aria-controls={featuresId}
        onClick={() => setOpen((value) => !value)}
        className="group inline-flex max-w-full items-center gap-1 justify-self-start rounded-sm text-left text-sm font-medium text-foreground"
      >
        <span className="truncate">{addOn.name}</span>
        <ChevronRightIcon
          aria-hidden
          className="size-3 shrink-0 text-muted-foreground transition-transform duration-200 group-aria-expanded:rotate-90"
        />
      </Button>
      {side ? (
        <span className="col-start-2 row-span-2 row-start-1 flex items-center gap-1.5">{side}</span>
      ) : null}
      <p
        id={featuresId}
        className={cn(
          "col-start-1 text-xs text-muted-foreground",
          open ? "animate-rise whitespace-normal" : "truncate",
        )}
      >
        {formatList(addOn.features.map((feature) => feature.name))}
      </p>
    </li>
  );
}

function AddOnList({
  addOns,
  onPlan,
  manageUrl,
}: {
  addOns: ManagedPlanAddOn[];
  onPlan: boolean;
  manageUrl: string | null;
}) {
  return (
    <ul className="grid grid-cols-1 sm:grid-cols-2">
      {addOns.map((addOn) => (
        <AddOnRow key={addOn.key} addOn={addOn} onPlan={onPlan} manageUrl={manageUrl} />
      ))}
    </ul>
  );
}

function ChangeLines({ change }: { change: ManagedPlanChange }) {
  const t = useT();
  const lines: { key: string; sentences: string[] }[] = [];

  if (change.starting) {
    lines.push({
      key: "starting",
      sentences: [
        change.joined.length > 0
          ? t("Your plan started with {0}.", formatList(change.joined.map((addOn) => addOn.name)))
          : t("Your plan started with the base plan alone."),
      ],
    });
  } else if (change.joined.length > 0) {
    lines.push({
      key: "joined",
      sentences: [t("{0} joined your plan.", formatList(change.joined.map((addOn) => addOn.name)))],
    });
  }
  if (change.left.length > 0) {
    lines.push({
      key: "left",
      sentences: [
        t("{0} left your plan.", formatList(change.left.map((addOn) => addOn.name))),
        t(
          "{0, plural, one {Everything entered with it is kept and comes back if you add it again.} other {Everything entered with them is kept and comes back if you add them again.}}",
          change.left.length,
        ),
      ],
    });
  }
  if (change.ending.length > 0) {
    lines.push({
      key: "ending",
      sentences: [
        ...change.ending.map((entry) =>
          t("{0} was set to end on {1}.", entry.addOn.name, formatUnixMonthDay(entry.endsAt)),
        ),
        t(
          "{0, plural, one {Everything entered with it is kept and comes back if you add it again.} other {Everything entered with them is kept and comes back if you add them again.}}",
          change.ending.length,
        ),
      ],
    });
  }
  if (change.kept.length > 0) {
    lines.push({
      key: "kept",
      sentences: [
        t(
          "{1} {0, plural, one {stays on your plan.} other {stay on your plan.}}",
          change.kept.length,
          formatList(change.kept.map((addOn) => addOn.name)),
        ),
      ],
    });
  }

  return (
    <div className="flex min-w-0 flex-col gap-1">
      {lines.map((line) => (
        <p key={line.key} className="text-foreground">
          {line.sentences.map((sentence, index) => (
            <Fragment key={sentence}>
              {index > 0 ? " " : null}
              {sentence}
            </Fragment>
          ))}
        </p>
      ))}
    </div>
  );
}

function ChangeRow({ change }: { change: ManagedPlanChange }) {
  return (
    <li className="grid grid-cols-[96px_minmax(0,1fr)] gap-x-3 gap-y-1 px-3 py-2.5 text-xs">
      <time
        dateTime={new Date(change.changedAt * 1000).toISOString()}
        className="text-muted-foreground tabular-nums"
      >
        {formatUnixDateMedium(change.changedAt)}
      </time>
      <ChangeLines change={change} />
    </li>
  );
}

/**
 * The plan as its administrator reads it: how much of it is used this period, the add-ons on it
 * and off it, what every plan includes, and the changes that moved an add-on.
 */
export function PlanSections({ plan }: { plan: ManagedPlan }) {
  const t = useT();
  const manageUrl = plan.manageUrl ?? null;

  return (
    <div className="flex flex-col gap-4">
      <PlanEndNotice
        endsAt={plan.endsAt}
        finalBackupDays={plan.finalBackupDays}
        manageUrl={manageUrl}
      />
      {plan.usage ? <PlanUsage usage={plan.usage} /> : null}

      <SectionPanel title={t("On your plan")} count={plan.included.length}>
        {plan.included.length > 0 ? (
          <AddOnList addOns={plan.included} onPlan manageUrl={manageUrl} />
        ) : (
          <SectionPanelQuiet>{t("Your plan has no add-ons.")}</SectionPanelQuiet>
        )}
      </SectionPanel>

      <SectionPanel title={t("You can add")} count={plan.available.length}>
        {plan.available.length > 0 ? (
          <AddOnList addOns={plan.available} onPlan={false} manageUrl={manageUrl} />
        ) : (
          <SectionPanelQuiet>{t("Every add-on is on your plan.")}</SectionPanelQuiet>
        )}
      </SectionPanel>

      {plan.base.length > 0 ? (
        <SectionPanel title={t("Included in every plan")}>
          <p className="max-w-[92ch] px-3 py-3 text-xs text-foreground">
            {formatList(plan.base.map((feature) => feature.name))}
          </p>
        </SectionPanel>
      ) : null}

      <SectionPanel title={t("Recent changes")}>
        {plan.changes.length > 0 ? (
          <ul className="flex flex-col divide-y divide-border-subtle">
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
