import { useT } from "@trenova/shared/i18n/use-t";
import { Figs, type Fig } from "../kit/layout";
import type { ActivityFacts } from "./activity-model";

/** Today's runs, how many failed, what waits on a decision, and how often people agree. */
export function ActivityFigures({ facts }: { facts: ActivityFacts }) {
  const t = useT();
  const items: Fig[] = [
    {
      label: t("Runs today"),
      value: facts.runs.toLocaleString(),
      sub:
        facts.working > 0
          ? t("{0, plural, one {# running now} other {# running now}}", facts.working)
          : t("none running now"),
    },
    {
      label: t("Failed"),
      value: facts.failed.toLocaleString(),
      sub: t("of today's runs"),
      tone: facts.failed > 0 ? "t-d" : undefined,
    },
  ];
  if (facts.pending !== null) {
    items.push(
      {
        label: t("Awaiting a decision"),
        value: facts.pending.toLocaleString(),
        sub: t("proposals"),
        tone: facts.pending > 0 ? "t-w" : undefined,
      },
      {
        label: t("Approved as proposed"),
        value:
          facts.approvedAsProposed === null
            ? "—"
            : `${Math.round(facts.approvedAsProposed * 100)}%`,
        sub: t("{0, plural, one {last # day} other {last # days}}", facts.decisionWindowDays),
      },
    );
  }

  return <Figs items={items} label={t("Activity figures")} />;
}
