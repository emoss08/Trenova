import type { WatchtowerClockPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { formatFactTime } from "../../watchtower/lanes";
import { Figure, Figures, PanelBody, PanelCard } from "./panel-parts";

/**
 * A detention clock as a meter: the free time the customer was given, then
 * the time that bills. It reads as it is billed, free first and billable
 * after, with what has accrued beside it.
 */
export function ClockPanel({
  clock,
  now,
  timezone,
}: {
  clock: WatchtowerClockPanel;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const billing = clock.billableMinutes > 0;
  const total = Math.max(clock.freeMinutes + clock.billableMinutes, 1);
  const freeShare = Math.max(6, (clock.freeMinutes / total) * 100);
  const there = clock.departedAt <= 0;

  return (
    <PanelCard
      eyebrow={clock.customer ? t("Detention · {0}", clock.customer) : t("Detention")}
      title={clock.place}
      badge={
        there ? (
          <Badge variant={billing ? "danger" : "warning"} dot>
            {billing ? t("Billing") : t("On free time")}
          </Badge>
        ) : (
          <Badge variant="neutral" dot>
            {t("Departed")}
          </Badge>
        )
      }
    >
      <PanelBody>
        <div
          className="mt-1 flex h-2 overflow-hidden rounded-full bg-dsk-sunken"
          role="img"
          aria-label={t(
            "{0} free, then {1} billable",
            formatMinutesSpan(clock.freeMinutes),
            formatMinutesSpan(clock.billableMinutes),
          )}
        >
          <span
            className="h-full bg-dsk-success"
            style={{ width: `${billing ? freeShare : 100}%` }}
          />
          {billing && <span className="h-full flex-1 bg-dsk-error" />}
        </div>
        <div className="mt-1.5 flex justify-between text-xs text-dsk-subtle">
          <span>{t("Free {0}", formatMinutesSpan(clock.freeMinutes))}</span>
          {billing && <span>{t("Billable {0}", formatMinutesSpan(clock.billableMinutes))}</span>}
        </div>
      </PanelBody>
      <Figures>
        {clock.arrivedAt > 0 && (
          <Figure label={t("Arrived")} value={formatFactTime(t, clock.arrivedAt, now, timezone)} />
        )}
        {clock.freeEndsAt > 0 && (
          <Figure
            label={billing ? t("Free time ended") : t("Free time ends")}
            value={formatFactTime(t, clock.freeEndsAt, now, timezone)}
          />
        )}
        {clock.departedAt > 0 && (
          <Figure
            label={t("Departed")}
            value={formatFactTime(t, clock.departedAt, now, timezone)}
          />
        )}
        {billing && (
          <Figure
            label={t("Billable so far")}
            value={clock.accrued || formatMinutesSpan(clock.billableMinutes)}
            note={clock.accrued ? formatMinutesSpan(clock.billableMinutes) : undefined}
            tone="bad"
          />
        )}
      </Figures>
    </PanelCard>
  );
}
