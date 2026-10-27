import { useT } from "@trenova/shared/i18n/use-t";
import { RunHistorySkeleton } from "../extensions/run-history-skeleton";

/** The analysis history while it loads. */
export function AnalysisRunsSkeleton() {
  const t = useT();

  return <RunHistorySkeleton label={t("Loading analyses")} />;
}
