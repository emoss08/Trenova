import { useT } from "@trenova/shared/i18n/use-t";
import { RunHistorySkeleton } from "./run-history-skeleton";

/** The web watches while they load. */
export function WebWatchesSkeleton() {
  const t = useT();

  return <RunHistorySkeleton label={t("Loading web watches")} />;
}
