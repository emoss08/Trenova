import type { WatchtowerRunPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelBody, PanelCard } from "./panel-parts";

/** An agent run that could not finish: which agent, when it stopped, and why in its own words. */
export function RunPanel({
  run,
  now,
  timezone,
}: {
  run: WatchtowerRunPanel;
  now: number;
  timezone: string;
}) {
  const t = useT();

  return (
    <PanelCard
      eyebrow={
        run.failedAt > 0
          ? t("Agent run · stopped {0}", formatFactTime(t, run.failedAt, now, timezone))
          : t("Agent run")
      }
      title={run.agent}
      badge={
        <Badge variant="danger" dot>
          {t("Could not finish")}
        </Badge>
      }
    >
      <PanelBody>
        <p className="text-dsk-base text-dsk-fg2">
          {run.reason || t("It stopped without saying why. Its run shows how far it got.")}
        </p>
      </PanelBody>
    </PanelCard>
  );
}
