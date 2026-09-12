import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import type { TeamSummary } from "@/lib/my-team";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { formatTenure } from "@trenova/shared/lib/tenure";
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, HourglassIcon, ShieldCheckIcon, UsersIcon } from "lucide-react";

const DAY_SECONDS = 86_400;

type TeamSummaryStripProps = {
  summary: TeamSummary;
  now: number;
};

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

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
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label={t("On your team")}
          info={
            <InfoPopover title={t("On your team")}>
              {t(
                "Everyone whose approvals reach you: your own reports, people at a terminal you run, and anyone you are covering for under a delegation in force today.",
              )}
            </InfoPopover>
          }
        />
        <NumberFlow value={summary.total} className={VALUE_CLASS} aria-label={t("On your team")} />
        <CompositionBar
          segments={segments}
          size="sm"
          className="mt-auto"
          aria-label={t("How they reach you")}
        />
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<ShieldCheckIcon className="size-[11px]" />}
          label={t("In good standing")}
          info={
            <InfoPopover title={t("In good standing")}>
              {t(
                "People with nothing critical and nothing on watch: compliance in order, training current, safety rating Excellent or Good.",
              )}
            </InfoPopover>
          }
        />
        <div className="flex items-center gap-2.5">
          <RingGauge
            value={goodShare}
            size={40}
            strokeWidth={4}
            tone="brand"
            aria-label={t("Share of the team in good standing")}
          />
          <div className="flex items-baseline gap-1">
            <NumberFlow value={summary.goodStanding} className={VALUE_CLASS} />
            <span className="text-muted-foreground font-mono text-[11px]">
              {t("of {0}", summary.total)}
            </span>
          </div>
        </div>
        <KpiSub>{t("Compliant, trained and rated Good or better")}</KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<AlertTriangleIcon className="size-[11px]" />}
          label={t("Needing attention")}
          info={
            <InfoPopover title={t("Needing attention")}>
              {t(
                "Anyone with something that stops them working or should: non-compliant, training that blocks dispatch, or a safety rating of At risk. Watch items alone do not count here.",
              )}
            </InfoPopover>
          }
        />
        <NumberFlow
          value={summary.attention}
          className={cn(VALUE_CLASS, summary.attention > 0 && "text-destructive")}
          aria-label={t("Needing attention")}
        />
        <KpiSub>{describeAttention(summary)}</KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<HourglassIcon className="size-[11px]" />}
          label={t("Average tenure")}
          info={
            <InfoPopover title={t("Average tenure")}>
              {t(
                "Mean time since hire date across the team, for people with a hire date on file. A leaver is measured to their termination date.",
              )}
            </InfoPopover>
          }
        />
        <span className={VALUE_CLASS}>{averageTenure}</span>
        <KpiSub>
          {summary.longestServing
            ? t(
                "Longest serving: {0}, {1}",
                summary.longestServing.member.name,
                formatTenure(now - summary.longestServing.days * DAY_SECONDS, null, now),
              )
            : t("No hire dates on record")}
        </KpiSub>
      </KpiCard>
    </div>
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
