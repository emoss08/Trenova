import { useT } from "@trenova/shared/i18n/use-t";
import { RunHistorySkeleton } from "./run-history-skeleton";

/** The portal task history while it loads. */
export function PortalRunsSkeleton() {
  const t = useT();

  return <RunHistorySkeleton label={t("Loading portal tasks")} />;
}
