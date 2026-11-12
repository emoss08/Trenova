import type { WatchtowerCoveragePanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { Figure, Figures, PanelBody, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

/**
 * A move with nobody on it: when it leaves, and why it is uncovered. A tender
 * every carrier passed on is the same trouble one step further along.
 */
export function CoveragePanel({
  coverage,
  now,
  timezone,
}: {
  coverage: WatchtowerCoveragePanel;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const left = coverage.startsAt > 0 && coverage.startsAt <= now;

  return (
    <PanelCard
      eyebrow={t("Coverage")}
      title={
        coverage.exhausted
          ? t("Every carrier in the routing guide passed")
          : left
            ? t("Should have left with nobody on it")
            : t("Nobody is on this move")
      }
      badge={
        <Badge variant="danger" dot>
          {coverage.exhausted ? t("Tender exhausted") : t("Uncovered")}
        </Badge>
      }
    >
      {coverage.reason !== "" && (
        <PanelBody>
          <p className="text-dsk-base text-dsk-fg2">{coverage.reason}</p>
        </PanelBody>
      )}
      <Figures>
        {coverage.startsAt > 0 && (
          <Figure
            label={left ? t("Was due to leave") : t("Leaves")}
            value={formatFactTime(t, coverage.startsAt, now, timezone)}
            note={agoOrIn(t, coverage.startsAt, now)}
            tone={left ? "bad" : "warn"}
          />
        )}
        {coverage.endedAt > 0 && (
          <Figure
            label={t("Last carrier passed")}
            value={formatFactTime(t, coverage.endedAt, now, timezone)}
            note={agoOrIn(t, coverage.endedAt, now)}
          />
        )}
      </Figures>
    </PanelCard>
  );
}
