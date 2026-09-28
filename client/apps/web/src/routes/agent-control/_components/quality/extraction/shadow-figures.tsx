import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { ExtractionShadowReport } from "@/lib/graphql/extraction-shadow";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShare, formatUsd } from "../quality-model";
import { accuracyTone } from "./extraction-model";
import { accuracyGap } from "./shadow-model";

/**
 * The shadow comparison in figures: both sides' accuracy on the same confirmed
 * documents, how often the candidate did better or worse, how many shadows ran,
 * and what they cost.
 */
export function ShadowFigures({ report }: { report: ExtractionShadowReport }) {
  const t = useT();
  const gap = accuracyGap(report.candidate, report.production);

  return (
    <KpiStrip aria-label={t("Shadow comparison figures")}>
      <KpiStripItem
        label={t("Candidate accuracy")}
        value={report.candidate.scored > 0 ? formatShare(report.candidate.accuracy) : "—"}
        sub={
          gap === null
            ? t("against production")
            : gap > 0
              ? t("{0} pts above production", gap)
              : gap < 0
                ? t("{0} pts below production", -gap)
                : t("level with production")
        }
        tone={accuracyTone(report.candidate.accuracy, report.candidate.scored)}
      />
      <KpiStripItem
        label={t("Production accuracy")}
        value={report.production.scored > 0 ? formatShare(report.production.accuracy) : "—"}
        sub={t(
          "{0, plural, one {on # confirmed document} other {on # confirmed documents}}",
          report.scored,
        )}
        tone={accuracyTone(report.production.accuracy, report.production.scored)}
      />
      <KpiStripItem
        label={t("Better or worse")}
        value={t("{0} better, {1} worse", report.better, report.worse)}
        sub={t("{0, plural, one {# the same} other {# the same}}", report.same)}
        tone={
          report.scored === 0 ? undefined : report.worse > report.better ? "warning" : "success"
        }
      />
      <KpiStripItem
        label={t("Shadowed")}
        value={report.sampled}
        sub={t(
          "{0} failed, {1} skipped, {2} pending",
          report.failed,
          report.skipped,
          report.pending,
        )}
        tone={report.failed > 0 ? "warning" : undefined}
      />
      <KpiStripItem
        label={t("Cost")}
        value={formatUsd(report.costUsd)}
        sub={
          report.avgLatencyMs > 0
            ? t("{0} ms on average", report.avgLatencyMs)
            : t("from the evaluation budget")
        }
      />
    </KpiStrip>
  );
}
