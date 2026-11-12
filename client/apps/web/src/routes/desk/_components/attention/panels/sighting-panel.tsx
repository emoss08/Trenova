import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { Figure, Figures, PanelBody, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

type SightingPanelData = NonNullable<WatchtowerPanel["sighting"]>;

/**
 * A truck the tracker saw at a stop with nothing recorded for it: where,
 * when, and why Trenova did not record it on its own.
 */
export function SightingPanel({
  sighting,
  now,
  timezone,
}: {
  sighting: SightingPanelData;
  now: number;
  timezone: string;
}) {
  const t = useT();

  return (
    <PanelCard
      eyebrow={sighting.departure ? t("Seen leaving") : t("Seen arriving")}
      title={sighting.place || t("A stop on this load")}
      badge={
        <Badge variant="warning" dot>
          {t("Not recorded")}
        </Badge>
      }
    >
      {sighting.why !== "" && (
        <PanelBody>
          <p className="text-dsk-base text-dsk-fg2">{sighting.why}</p>
        </PanelBody>
      )}
      {sighting.seenAt > 0 && (
        <Figures>
          <Figure
            label={t("The tracker saw it")}
            value={formatFactTime(t, sighting.seenAt, now, timezone)}
            note={agoOrIn(t, sighting.seenAt, now)}
          />
        </Figures>
      )}
    </PanelCard>
  );
}
