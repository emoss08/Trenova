import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DueCheck } from "../../record-card/card-due";
import { CardIcon, type CardCheckState, type CardFact } from "../../record-card/record-card";
import { cardDay } from "../../record-card/record-card-values";
import {
  CALENDAR_AHEAD_DAYS,
  CALENDAR_BACK_DAYS,
  calendarPlace,
  type CalendarMark,
} from "./trailer-card";

const DAY_SECONDS = 86_400;
/** How near either end of the axis a mark's flag turns inward, so it stays on the card. */
const EDGE = 0.16;

const STATE_ICON: Record<CardCheckState, "check" | "warn" | "x" | "info"> = {
  ok: "check",
  warn: "warn",
  bad: "x",
  info: "info",
};

/** The first of each month inside the calendar's span, with where it falls along it. */
function monthTicks(now: number): { key: string; label: string; share: number }[] {
  const start = new Date((now - CALENDAR_BACK_DAYS * DAY_SECONDS) * 1000);
  const end = (now + CALENDAR_AHEAD_DAYS * DAY_SECONDS) * 1000;
  const ticks: { key: string; label: string; share: number }[] = [];
  const cursor = new Date(start.getFullYear(), start.getMonth() + 1, 1);
  while (cursor.getTime() <= end) {
    const at = Math.floor(cursor.getTime() / 1000);
    ticks.push({
      key: `${cursor.getFullYear()}-${cursor.getMonth()}`,
      label: cursor.toLocaleDateString(undefined, {
        month: "short",
        ...(cursor.getMonth() === 0 ? { year: "2-digit" } : {}),
      }),
      share: calendarPlace(at, now).share,
    });
    cursor.setMonth(cursor.getMonth() + 1);
  }
  return ticks;
}

/**
 * The trailer's service calendar: a season back and a year ahead on one axis,
 * today marked, and each date its papers hang on placed where it falls,
 * coloured by how it stands; under it, what each check says, worst first. A
 * trailer out of service says so above the axis, since that has no date.
 */
export function ServiceCalendar({
  marks,
  checks,
  now,
}: {
  marks: readonly CalendarMark[];
  checks: readonly DueCheck[];
  now: number;
}) {
  const t = useT();
  const today = calendarPlace(now, now).share;
  return (
    <div className="dk-tl-cal">
      <div className="dk-tl-axis" role="img" aria-label={t("Service calendar")}>
        <div className="dk-tl-months" aria-hidden>
          {monthTicks(now).map((tick) => (
            <span key={tick.key} style={{ left: `${tick.share * 100}%` }}>
              {tick.label}
            </span>
          ))}
        </div>
        <div className="dk-tl-rail">
          <i className="dk-tl-past" style={{ width: `${today * 100}%` }} />
          <span className="dk-tl-today" style={{ left: `${today * 100}%` }}>
            <em>{t("Today")}</em>
          </span>
          {marks.map((mark, index) => {
            const place = calendarPlace(mark.at, now);
            return (
              <span
                key={mark.key}
                className={cn(
                  "dk-tl-mark",
                  `dk-${mark.state}`,
                  index % 2 === 1 && "dk-low",
                  place.outside && `dk-${place.outside}`,
                  place.share < EDGE ? "dk-start" : place.share > 1 - EDGE && "dk-end",
                )}
                style={{ left: `${place.share * 100}%` }}
              >
                <i aria-hidden />
                <b>{mark.label}</b>
                <span>{cardDay(mark.at)}</span>
              </span>
            );
          })}
        </div>
      </div>
      <ul className="dk-tl-legend">
        {checks.map((check) => (
          <li key={check.key} className={`dk-${check.state}`}>
            <CardIcon name={STATE_ICON[check.state]} size={11} stroke={2.6} />
            <span>
              <b>{check.title}</b>
              <span>{check.detail}</span>
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** The trailer's own facts stamped on a data plate, as the plate riveted to its frame reads. */
export function DataPlate({ facts }: { facts: readonly CardFact[] }) {
  return (
    <dl className="dk-tl-plate">
      {facts.map((fact) => (
        <div key={fact.key}>
          <dt>{fact.label}</dt>
          <dd>{fact.value}</dd>
        </div>
      ))}
    </dl>
  );
}
