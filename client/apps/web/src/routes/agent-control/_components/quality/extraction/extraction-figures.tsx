import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { ExtractionAccuracy } from "@/lib/graphql/extraction-eval";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShare } from "../quality-model";
import { accuracyTone, modelLabel } from "./extraction-model";

/**
 * Document extraction in four figures: how often production reads a field
 * right, how many corrections that rests on, how big the evaluation set is,
 * and what the latest evaluation run scored.
 */
export function ExtractionFigures({ accuracy }: { accuracy: ExtractionAccuracy }) {
  const t = useT();
  const latest = accuracy.recentRuns[0];

  return (
    <KpiStrip aria-label={t("Document extraction figures")}>
      <KpiStripItem
        label={t("Field accuracy")}
        value={accuracy.scored > 0 ? formatShare(accuracy.accuracy) : "—"}
        sub={t(
          "{0, plural, one {in production, last # day} other {in production, last # days}}",
          accuracy.windowDays,
        )}
        tone={accuracyTone(accuracy.accuracy, accuracy.scored)}
      />
      <KpiStripItem
        label={t("Corrections")}
        value={accuracy.corrections}
        sub={
          accuracy.sampled
            ? t("most recent only")
            : t("{0, plural, one {# field scored} other {# fields scored}}", accuracy.scored)
        }
      />
      <KpiStripItem
        label={t("Evaluation set")}
        value={accuracy.cases.active}
        sub={t("{0, plural, one {# candidate} other {# candidates}}", accuracy.cases.candidate)}
        tone={accuracy.cases.active === 0 ? "warning" : undefined}
      />
      <KpiStripItem
        label={t("Latest run")}
        value={latest ? formatShare(latest.accuracy) : "—"}
        sub={latest ? modelLabel(latest.servedModel || latest.providerModel, t) : t("No runs yet")}
        tone={latest ? accuracyTone(latest.accuracy, latest.scoredCount) : undefined}
      />
    </KpiStrip>
  );
}
