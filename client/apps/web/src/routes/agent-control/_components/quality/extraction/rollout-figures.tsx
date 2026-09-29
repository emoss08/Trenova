import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { ExtractionRolloutReport } from "@/lib/graphql/extraction-rollout";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShare } from "../quality-model";
import { accuracyTone } from "./extraction-model";
import { accuracyGap } from "./shadow-model";

/**
 * The rollout in figures: how the candidate reads the documents it serves
 * beside production on the rest, how often its answers are unusable, and how
 * many documents it was sent.
 */
export function RolloutFigures({ report }: { report: ExtractionRolloutReport }) {
  const t = useT();
  const gap = accuracyGap(report.candidateAccuracy, report.productionAccuracy);
  const candidateSettled =
    report.candidate.accepted + report.candidate.rejected + report.candidate.failed;
  const rejectionGap = Math.round(
    (report.candidate.rejectionRate - report.control.rejectionRate) * 100,
  );

  return (
    <KpiStrip aria-label={t("Rollout figures")}>
      <KpiStripItem
        label={t("Candidate accuracy")}
        value={
          report.candidateAccuracy.scored > 0 ? formatShare(report.candidateAccuracy.accuracy) : "—"
        }
        sub={
          gap === null
            ? t("against production")
            : gap > 0
              ? t("{0} pts above production", gap)
              : gap < 0
                ? t("{0} pts below production", -gap)
                : t("level with production")
        }
        tone={accuracyTone(report.candidateAccuracy.accuracy, report.candidateAccuracy.scored)}
      />
      <KpiStripItem
        label={t("Production accuracy")}
        value={
          report.productionAccuracy.scored > 0
            ? formatShare(report.productionAccuracy.accuracy)
            : "—"
        }
        sub={t(
          "{0, plural, one {on # confirmed field} other {on # confirmed fields}}",
          report.productionAccuracy.scored,
        )}
        tone={accuracyTone(report.productionAccuracy.accuracy, report.productionAccuracy.scored)}
      />
      <KpiStripItem
        label={t("Unusable answers")}
        value={candidateSettled > 0 ? formatShare(report.candidate.rejectionRate) : "—"}
        sub={t("{0} for production", formatShare(report.control.rejectionRate))}
        tone={candidateSettled > 0 && rejectionGap > 0 ? "warning" : undefined}
      />
      <KpiStripItem
        label={t("Sent to the candidate")}
        value={report.candidate.assigned}
        sub={t(
          "{0} fell back to production, {1} pending",
          report.candidate.fellBack,
          report.candidate.pending,
        )}
        tone={report.candidate.fellBack > 0 ? "warning" : undefined}
      />
      <KpiStripItem
        label={t("Kept on production")}
        value={report.control.assigned}
        sub={t("{0}% of documents to the candidate", report.rollout.percent)}
      />
    </KpiStrip>
  );
}
