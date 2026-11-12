import type { WatchtowerContactPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { Figure, Figures, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

/**
 * A carrier that has gone quiet: when it was last heard from, how often it is
 * meant to check in, and how many check calls it has now missed.
 */
export function ContactPanel({
  contact,
  now,
  timezone,
}: {
  contact: WatchtowerContactPanel;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const every = contact.everyHours * 3600;
  const missed =
    every > 0 && contact.lastHeardAt > 0
      ? Math.max(1, Math.floor((now - contact.lastHeardAt) / every))
      : 0;

  return (
    <PanelCard
      eyebrow={t("Check calls")}
      title={contact.who}
      badge={
        <Badge variant={missed > 1 ? "danger" : "warning"} dot>
          {missed > 0 ? t("{0, plural, one {# missed} other {# missed}}", missed) : t("Silent")}
        </Badge>
      }
    >
      <Figures>
        {contact.lastHeardAt > 0 && (
          <Figure
            label={t("Last heard from")}
            value={formatFactTime(t, contact.lastHeardAt, now, timezone)}
            note={agoOrIn(t, contact.lastHeardAt, now)}
          />
        )}
        {contact.everyHours > 0 && (
          <Figure
            label={t("Expected")}
            value={t("{0, plural, one {Every hour} other {Every # hours}}", contact.everyHours)}
          />
        )}
        {contact.dueAt > 0 && (
          <Figure
            label={t("Was due")}
            value={formatFactTime(t, contact.dueAt, now, timezone)}
            note={agoOrIn(t, contact.dueAt, now)}
            tone="bad"
          />
        )}
      </Figures>
    </PanelCard>
  );
}
