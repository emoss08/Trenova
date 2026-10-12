import type { LocationDwell } from "@/lib/graphql/location-card";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatPercent } from "@trenova/shared/lib/utils";
import { CardIcon, type CardFact } from "../../record-card/record-card";
import { waitText } from "../../record-card/record-card-values";
import { dwellMoneyLine, dwellScale, type Amenity } from "./location-card";

/** A share of stops past free time at or above this reads as a dock worth watching. */
const BREACH_WARN_SHARE = 0.25;

/** What the facility offers and asks, a chip each, ticked when it is so and crossed when not. */
export function AmenityChips({ items }: { items: readonly Amenity[] }) {
  if (items.length === 0) return null;
  return (
    <ul className="dk-lo-amen">
      {items.map((item) => (
        <li key={item.key} className={cn(!item.on && "dk-off")}>
          <CardIcon name={item.on ? "check" : "x"} size={10} stroke={2.8} />
          {item.label}
        </li>
      ))}
    </ul>
  );
}

/**
 * How long trucks wait at the dock, on a ruler from no wait to a little past
 * the longest: the half that wait less than the median filled solid, the
 * slower stretch to the 90th percentile hatched, the average marked, and how
 * many stops ran past their free time beside it.
 */
export function DwellRuler({ dwell }: { dwell: LocationDwell }) {
  const t = useT();
  const scale = dwellScale(dwell, t);
  const median = scale.marks.find((mark) => mark.key === "median")?.share ?? 0;
  const p90 = scale.marks.find((mark) => mark.key === "p90")?.share ?? 0;
  const breachShare = dwell.stopCount > 0 ? dwell.breachCount / dwell.stopCount : 0;
  return (
    <div className="dk-lo-dwell">
      <div className="dk-lo-ruler" role="img" aria-label={t("Dwell distribution")}>
        <div className="dk-lo-bar">
          <i className="dk-lo-half" style={{ width: `${median * 100}%` }} />
          <i
            className="dk-lo-tail"
            style={{ left: `${median * 100}%`, width: `${Math.max(0, p90 - median) * 100}%` }}
          />
          {scale.marks.map((mark) => (
            <span
              key={mark.key}
              className={cn("dk-lo-pin", `dk-${mark.key}`)}
              style={{ left: `${mark.share * 100}%` }}
            >
              <b>{waitText(mark.minutes)}</b>
              <em>{mark.label}</em>
            </span>
          ))}
        </div>
        <div className="dk-lo-scale" aria-hidden>
          <span>{waitText(0)}</span>
          <span>{waitText(scale.maxMinutes / 2)}</span>
          <span>{waitText(scale.maxMinutes)}</span>
        </div>
      </div>
      <div className={cn("dk-lo-breach", breachShare >= BREACH_WARN_SHARE && "dk-warn")}>
        <b>{formatPercent(breachShare * 100, 0)}</b>
        <span>
          {t(
            "{0} of {1, plural, one {# stop} other {# stops}} ran past free time",
            formatNumber(dwell.breachCount),
            dwell.stopCount,
          )}
        </span>
        <i aria-hidden>
          <i style={{ width: `${Math.round(breachShare * 100)}%` }} />
        </i>
      </div>
      <p className="dk-rec-note dk-lo-money">{dwellMoneyLine(dwell, t)}</p>
    </div>
  );
}

/**
 * What is on file for the location as one address card: its street and town
 * set as an address, and the rest (code, category, timezone, coordinates,
 * when it was added) read down beside it, with its notes under them.
 */
export function LocationFile({
  street,
  city,
  notes,
  facts,
}: {
  street: string;
  city: string;
  /** The location's notes, or "" with none. */
  notes: string;
  facts: readonly CardFact[];
}) {
  const t = useT();
  return (
    <div className="dk-lo-file">
      <address>
        {street ? <b>{street}</b> : <b className="dk-off">{t("No street address on file")}</b>}
        {city && <span>{city}</span>}
      </address>
      <dl>
        {facts
          .filter((fact) => fact.key !== "description")
          .map((fact) => (
            <div key={fact.key}>
              <dt>{fact.label}</dt>
              <dd>{fact.value}</dd>
            </div>
          ))}
      </dl>
      {notes && <blockquote>{notes}</blockquote>}
    </div>
  );
}
