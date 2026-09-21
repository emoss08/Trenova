import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader } from "@/components/kpi/kpi-card";
import { KPI_VALUE_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { TeamSummary } from "@/lib/my-team";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { formatTenure } from "@trenova/shared/lib/tenure";

const DAY_SECONDS = 86_400;

type TeamSummaryStripProps = {
  summary: TeamSummary;
  now: number;
};

export function TeamSummaryStrip({ summary, now }: TeamSummaryStripProps) {
  const t = useT();

  const goodShare = summary.total > 0 ? summary.goodStanding / summary.total : 0;
  const averageTenure =
    summary.averageTenureDays != null
      ? formatTenure(now - summary.averageTenureDays * DAY_SECONDS, null, now)
      : "—";

  const segments = [
    { key: "direct", label: t("Direct"), value: summary.direct },
    { key: "terminal", label: t("Terminal"), value: summary.terminal },
    ...(summary.covering > 0
      ? [{ key: "covering", label: t("Covering"), value: summary.covering }]
      : []),
  ];

  return (
    <KpiStrip>
      <KpiCard span={2}>
        <KpiHeader
          label={t("On your team")}
          info={
            <InfoPopover title={t("On your team")}>
              {t(
                "Everyone whose approvals reach you: your own reports, people at a terminal you run, and anyone you are covering for under a delegation in force today.",
              )}
            </InfoPopover>
          }
        />
        <NumberFlow
          value={summary.total}
          className={KPI_VALUE_CLASS}
          aria-label={t("On your team")}
        />
        <CompositionBar
          segments={segments}
          size="sm"
          className="mt-auto"
          aria-label={t("How they reach you")}
        />
      </KpiCard>

      <KpiStripItem
        label={t("In good standing")}
        info={
          <InfoPopover title={t("In good standing")}>
            {t(
              "People with nothing critical and nothing on watch: compliance in order, training current, safety rating Excellent or Good.",
            )}
          </InfoPopover>
        }
        value={
          <span className="flex items-center gap-2">
            <RingGauge
              value={goodShare}
              size={24}
              strokeWidth={3}
              tone="brand"
              aria-label={t("Share of the team in good standing")}
            />
            <span className="flex items-baseline gap-1">
              <NumberFlow value={summary.goodStanding} />
              <span className="text-muted-foreground text-xs font-normal">
                {t("of {0}", summary.total)}
              </span>
            </span>
          </span>
        }
        sub={t("Compliant, trained and rated Good or better")}
      />

      <KpiStripItem
        label={t("Needing attention")}
        tone={summary.attention > 0 ? "danger" : undefined}
        info={
          <InfoPopover title={t("Needing attention")}>
            {t(
              "Anyone with something that stops them working or should: non-compliant, training that blocks dispatch, or a safety rating of At risk. Watch items alone do not count here.",
            )}
          </InfoPopover>
        }
        value={<NumberFlow value={summary.attention} aria-label={t("Needing attention")} />}
        sub={describeAttention(summary)}
      />

      <KpiStripItem
        label={t("Average tenure")}
        info={
          <InfoPopover title={t("Average tenure")}>
            {t(
              "Mean time since hire date across the team, for people with a hire date on file. A leaver is measured to their termination date.",
            )}
          </InfoPopover>
        }
        value={averageTenure}
        sub={
          summary.longestServing
            ? t(
                "Longest serving: {0}, {1}",
                summary.longestServing.member.name,
                formatTenure(now - summary.longestServing.days * DAY_SECONDS, null, now),
              )
            : t("No hire dates on record")
        }
      />
    </KpiStrip>
  );
}

function describeAttention(summary: TeamSummary): string {
  if (summary.attention === 0) {
    return summary.watching > 0
      ? `Nobody blocked. ${summary.watching} to keep an eye on.`
      : "Nobody blocked, nobody on watch.";
  }
  const parts: string[] = [];
  if (summary.byReason.compliance > 0) parts.push(`${summary.byReason.compliance} compliance`);
  if (summary.byReason.training > 0) parts.push(`${summary.byReason.training} training`);
  if (summary.byReason.safety > 0) parts.push(`${summary.byReason.safety} safety`);
  return parts.join(" · ");
}
