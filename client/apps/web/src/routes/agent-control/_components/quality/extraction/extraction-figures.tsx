import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { ExtractionAccuracy } from "@/lib/graphql/extraction-accuracy";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShare } from "../quality-model";
import { accuracyTone } from "./extraction-model";

/**
 * Document extraction in two figures: how often production reads a field right,
 * and how many corrections that rests on.
 */
export function ExtractionFigures({ accuracy }: { accuracy: ExtractionAccuracy }) {
  const t = useT();

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
    </KpiStrip>
  );
}
