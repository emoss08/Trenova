import type { ExtractionEvalRefreezeProgress } from "@/lib/graphql/extraction-eval";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Progress } from "@trenova/shared/components/ui/progress";
import { useT } from "@trenova/shared/i18n/use-t";

/** How far the organization's re-conversion through Docling has got, and what it found. */
export function RefreezeProgress({ progress }: { progress: ExtractionEvalRefreezeProgress }) {
  const t = useT();
  const done = progress.frozen + progress.skipped + progress.failed;
  const counts = t(
    "{0} re-converted, {1} skipped, {2} failed",
    progress.frozen,
    progress.skipped,
    progress.failed,
  );

  if (progress.status === "Running") {
    return (
      <Alert variant="info" size="sm">
        <AlertTitle>
          {t(
            "Re-converting cases with Docling: {0} of {1, plural, one {# case} other {# cases}}",
            done,
            progress.total,
          )}
        </AlertTitle>
        <AlertDescription>
          <Progress
            value={done}
            max={Math.max(progress.total, 1)}
            aria-label={t("Re-conversion progress")}
          />
          <p>{counts}</p>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert variant={progress.status === "Failed" ? "destructive" : "info"} size="sm">
      <AlertTitle>
        {progress.status === "Failed"
          ? t("The last re-conversion with Docling stopped before it finished")
          : t("The last re-conversion with Docling finished")}
      </AlertTitle>
      <AlertDescription>
        <p>
          {t("{0, plural, one {# case} other {# cases}} asked for: {1}", progress.total, counts)}
        </p>
      </AlertDescription>
    </Alert>
  );
}
