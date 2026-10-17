import type { ExtractionRolloutReport } from "@/lib/graphql/extraction-rollout";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * Documents the rollout left out of its comparison. They are served by
 * production and assigned to neither side, so without this the rollout's
 * figures would stop moving for a Docling tenant with no reason given.
 */
export function RolloutExclusions({ report }: { report: ExtractionRolloutReport }) {
  const t = useT();
  const doclingInput =
    report.exclusions.find((exclusion) => exclusion.reason === "docling_input")?.count ?? 0;

  if (doclingInput === 0) {
    return null;
  }

  return (
    <Alert variant="info" size="sm">
      <AlertDescription>
        <p>
          {t(
            "{0, plural, one {# document excluded: read by Docling} other {# documents excluded: read by Docling}}",
            doclingInput,
          )}
        </p>
        <p>
          {t(
            "The candidate is not sent documents read by Docling until it is retrained on Docling's output. Production reads them, and they count for neither side.",
          )}
        </p>
      </AlertDescription>
    </Alert>
  );
}
