import type { AgentQualityOverview } from "@/lib/graphql/agent-quality";
import { useT } from "@trenova/shared/i18n/use-t";
import { Figs, type Fig } from "../kit/layout";
import { formatShare, formatUsd } from "./quality-model";

/**
 * The organization's AI quality in five figures: what people thought of the answers,
 * how many they rated, how the agents score against their golden sets, how many
 * regressed, and what evaluation has spent this month.
 */
export function QualityFigures({ overview }: { overview: AgentQualityOverview }) {
  const t = useT();
  const hidden = t("Needs access to ratings");
  const overBudget = Number(overview.evalSpendMonthUsd) >= Number(overview.monthlyBudgetUsd);

  const items: Fig[] = [
    {
      label: t("Satisfaction"),
      value: overview.ratingsVisible ? formatShare(overview.satisfaction) : "—",
      sub: overview.ratingsVisible
        ? t("{0, plural, one {last # day} other {last # days}}", overview.windowDays)
        : hidden,
    },
    {
      label: t("Ratings"),
      value: overview.ratingsVisible ? overview.ratings.toLocaleString() : "—",
      sub: overview.ratingsVisible ? t("thumbs up and down") : hidden,
    },
    {
      label: t("Quality score"),
      value: formatShare(overview.qualityScore),
      sub: t("{0, plural, one {# agent scored} other {# agents scored}}", overview.agentsScored),
    },
    {
      label: t("Regressions"),
      value: overview.openRegressions,
      sub:
        overview.openRegressions > 0
          ? t(
              "{0, plural, one {# agent still regressed} other {# agents still regressed}}",
              overview.openRegressions,
            )
          : t("none open"),
      tone: overview.openRegressions > 0 ? "t-d" : undefined,
    },
    {
      label: t("Eval spend this month"),
      value: formatUsd(overview.evalSpendMonthUsd),
      sub: t("of {0}", formatUsd(overview.monthlyBudgetUsd)),
      tone: overBudget ? "t-w" : undefined,
    },
  ];

  return <Figs items={items} label={t("AI quality figures")} />;
}
