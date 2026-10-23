import { formatList, formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { clampUnit, cn, formatPercent } from "@trenova/shared/lib/utils";
import { Fragment } from "react";
import { CardIcon } from "../../record-card/record-card";
import { cardMoment, money } from "../../record-card/record-card-values";
import {
  actualsLabel,
  assignmentStatusLabel,
  carrierStatusLabel,
  coverageSummary,
  freightClassLabel,
  initialsOf,
  marginGauge,
  punctualityLabel,
  stopTypeLabel,
  temperatureLabel,
  windowLabel,
  type Charges,
  type EtaReading,
  type FreightSummary,
  type JourneyStop,
  type Margin,
  type MoveCoverage,
  type NowReading,
  type StopPlace,
} from "./shipment-card";

/*
The shipment card's drawn parts: the live strip under the head, the stops down
their rail, who covers each move, the freight, the charges and the margin.
Each is given what the card worked out and draws it; none reads a query.
*/

function PlaceEnd({ place, end = false }: { place: StopPlace | null; end?: boolean }) {
  if (!place) return <span className={cn("dk-sh-end", end && "dk-to")} />;
  const label = place.label || place.name;
  return (
    <span className={cn("dk-sh-end", end && "dk-to")}>
      <b>{label}</b>
      {place.name && place.name !== label && <span>{place.name}</span>}
    </span>
  );
}

/**
 * The card's live strip: the lane end to end, the stops on it where they fall,
 * the part already driven filled in, and the truck where it is now, pulsing
 * softly while the shipment moves. Pointing at a stop here marks it on the
 * rail below, and the other way round. Under it, what is happening now.
 */
export function NowStrip({
  stops,
  offsets,
  position,
  from,
  to,
  live,
  closed,
  reading,
  eta,
  focus,
  onFocus,
}: {
  stops: readonly JourneyStop[];
  offsets: readonly number[];
  position: number;
  from: StopPlace | null;
  to: StopPlace | null;
  live: boolean;
  closed: boolean;
  reading: NowReading;
  eta: EtaReading | null;
  focus: string | null;
  onFocus: (stopId: string | null) => void;
}) {
  const t = useT();
  const at = `${clampUnit(position) * 100}%`;
  return (
    <div className={cn("dk-sh-now", live && "dk-sh-live", closed && "dk-sh-closed")}>
      {stops.length > 0 && (
        <>
          <div className="dk-sh-ends">
            <PlaceEnd place={from} />
            <PlaceEnd place={to} end />
          </div>
          <div
            className="dk-sh-track"
            role="img"
            aria-label={t("{0}% of the way", Math.round(clampUnit(position) * 100))}
          >
            <span className="dk-sh-done" style={{ width: at }} />
            {stops.map((stop, index) => (
              <span
                key={stop.id}
                className={cn("dk-sh-dot", `dk-${stop.phase}`, focus === stop.id && "dk-on")}
                style={{ left: `${(offsets[index] ?? 0) * 100}%` }}
                title={`${stopTypeLabel(stop.type, t)} · ${stop.place.label || stop.place.name}`}
                onMouseEnter={() => onFocus(stop.id)}
                onMouseLeave={() => onFocus(null)}
              />
            ))}
            {!closed && (
              <>
                <i className="dk-sh-ping" style={{ left: at }} />
                <span className="dk-sh-truck" style={{ left: at }}>
                  <CardIcon name="truck" size={13} stroke={2} />
                </span>
              </>
            )}
          </div>
        </>
      )}
      <div className={cn("dk-sh-read", `dk-sh-t-${reading.tone}`)}>
        <i aria-hidden />
        <div>
          <b>{reading.title}</b>
          {reading.detail && <span>{reading.detail}</span>}
        </div>
        {eta && (
          <span className={cn("dk-sh-chip", `dk-${eta.state}`)} title={eta.detail || undefined}>
            {eta.label}
            {eta.at !== null && <em>{t("ETA {0}", cardMoment(eta.at))}</em>}
          </span>
        )}
      </div>
    </div>
  );
}

function StopNode({ phase }: { phase: JourneyStop["phase"] }) {
  return (
    <span className="dk-sh-node" aria-hidden>
      {phase === "done" && <CardIcon name="check" size={10} stroke={3} />}
      {phase === "canceled" && <CardIcon name="x" size={9} stroke={2.8} />}
    </span>
  );
}

/**
 * Every stop in the order it is run, down a rail: what kind of stop and where,
 * its window, when the truck came and went, and whether it was on time. A
 * journey of several moves is broken where each move begins, naming who
 * covers it.
 */
export function StopRail({
  stops,
  coverage,
  focus,
  onFocus,
}: {
  stops: readonly JourneyStop[];
  coverage: readonly MoveCoverage[];
  focus: string | null;
  onFocus: (stopId: string | null) => void;
}) {
  const t = useT();
  const moves = coverage.length > 1;
  return (
    <ol className="dk-sh-stops">
      {stops.map((stop, index) => {
        const opens = moves && (index === 0 || stops[index - 1].moveIndex !== stop.moveIndex);
        const cover = opens ? coverage.find((entry) => entry.moveIndex === stop.moveIndex) : null;
        const name = stop.place.name || stop.place.label;
        const where = stop.place.name && stop.place.label !== name ? stop.place.label : "";
        const slot = windowLabel(stop, t);
        const actuals = actualsLabel(stop, t);
        return (
          <Fragment key={stop.id}>
            {opens && (
              <li className={cn("dk-sh-mv", cover?.kind === "none" && "dk-warn")}>
                <b>{t("Move {0}", stop.moveIndex + 1)}</b>
                {cover && <span>{coverageSummary(cover, t)}</span>}
              </li>
            )}
            <li
              className={cn("dk-sh-stop", `dk-${stop.phase}`, focus === stop.id && "dk-on")}
              onMouseEnter={() => onFocus(stop.id)}
              onMouseLeave={() => onFocus(null)}
            >
              <StopNode phase={stop.phase} />
              <div className="dk-sh-sb">
                <div className="dk-sh-sh">
                  <em className="dk-sh-tag">{stopTypeLabel(stop.type, t)}</em>
                  <b>{name}</b>
                  {stop.punctuality && (
                    <span className={cn("dk-sh-pun", `dk-${stop.punctuality.state}`)}>
                      {punctualityLabel(stop.punctuality, t)}
                    </span>
                  )}
                </div>
                {where && <span className="dk-sh-where">{where}</span>}
                {slot && <span className="dk-sh-win">{slot}</span>}
                {actuals && <span className="dk-sh-act">{actuals}</span>}
              </div>
            </li>
          </Fragment>
        );
      })}
    </ol>
  );
}

/** One move's cover: the driver and the equipment, the carrier and its cost, or nobody yet. */
export function CoverageRow({ coverage, numbered }: { coverage: MoveCoverage; numbered: boolean }) {
  const t = useT();
  const move = numbered ? (
    <em className="dk-sh-cvm">{t("Move {0}", coverage.moveIndex + 1)}</em>
  ) : null;

  if (coverage.kind === "none") {
    return (
      <div className="dk-sh-cv dk-none">
        <span className="dk-sh-av">
          <CardIcon name="warn" size={13} />
        </span>
        <div className="dk-sh-cvb">
          <b>
            {move}
            {t("No driver or carrier yet")}
          </b>
          <span>{t("Assign a driver or tender it to a carrier on the board")}</span>
        </div>
      </div>
    );
  }

  if (coverage.kind === "driver") {
    const equipment = [
      coverage.tractor ? t("Tractor {0}", coverage.tractor) : "",
      coverage.trailer ? t("Trailer {0}", coverage.trailer) : "",
    ].filter(Boolean);
    return (
      <div className="dk-sh-cv">
        <span className="dk-sh-av">{initialsOf(coverage.drivers[0] ?? "")}</span>
        <div className="dk-sh-cvb">
          <b>
            {move}
            {coverage.drivers.length > 0 ? formatList(coverage.drivers) : t("Driver assigned")}
          </b>
          {equipment.length > 0 && <span className="dk-sh-code">{equipment.join(" · ")}</span>}
        </div>
        <span className="dk-sh-cvs">{assignmentStatusLabel(coverage.status, t)}</span>
      </div>
    );
  }

  const details = [
    coverage.proNumber ? t("PRO {0}", coverage.proNumber) : "",
    coverage.driver ? t("Driver {0}", coverage.driver) : "",
  ].filter(Boolean);
  return (
    <div className="dk-sh-cv">
      <span className="dk-sh-av dk-co">{initialsOf(coverage.carrier)}</span>
      <div className="dk-sh-cvb">
        <b>
          {move}
          {coverage.carrier || t("Carrier assigned")}
          <span className={cn("dk-sh-pill", coverage.status === "Confirmed" ? "dk-ok" : "dk-wait")}>
            {carrierStatusLabel(coverage.status, t)}
          </span>
        </b>
        {details.length > 0 && <span className="dk-sh-code">{details.join(" · ")}</span>}
      </div>
      {coverage.cost !== null && (
        <span className="dk-sh-n">{money(coverage.cost, coverage.currency)}</span>
      )}
    </div>
  );
}

/** What is on the truck: the counts and conditions as chips, then each commodity line. */
export function FreightBlock({ freight }: { freight: FreightSummary }) {
  const t = useT();
  return (
    <>
      <div className="dk-sh-chips">
        {freight.pieces !== null && (
          <span className="dk-sh-fc">
            {t("{0, plural, one {# piece} other {# pieces}}", freight.pieces)}
          </span>
        )}
        {freight.weight !== null && (
          <span className="dk-sh-fc">{t("{0} lb", formatNumber(freight.weight))}</span>
        )}
        {freight.temperature && (
          <span className="dk-sh-fc">{temperatureLabel(freight.temperature, t)}</span>
        )}
        {freight.hazmat && (
          <span className="dk-sh-fc dk-warn">
            <CardIcon name="warn" size={11} stroke={2.2} />
            {t("Hazmat")}
          </span>
        )}
      </div>
      {freight.lines.length > 0 && (
        <div className="dk-sh-fl">
          {freight.lines.map((line) => {
            const amount = [
              line.pieces !== null ? t("{0} pcs", formatNumber(line.pieces)) : "",
              line.weight !== null ? t("{0} lb", formatNumber(line.weight)) : "",
            ].filter(Boolean);
            return (
              <div key={line.id} className="dk-sh-flr">
                <div>
                  <b>{line.name}</b>
                  {amount.length > 0 && <span>{amount.join(" · ")}</span>}
                </div>
                <span className="dk-sh-flags">
                  {freightClassLabel(line.freightClass, t) && (
                    <em>{freightClassLabel(line.freightClass, t)}</em>
                  )}
                  {line.hazmat && <em className="dk-warn">{t("Hazmat")}</em>}
                  {line.fragile && <em>{t("Fragile")}</em>}
                  {line.stackable === false && <em>{t("Not stackable")}</em>}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </>
  );
}

/** What the customer is charged, line by line to the total, and what holds the rate. */
export function ChargesBlock({ charges }: { charges: Charges }) {
  const t = useT();
  return (
    <>
      <div className="dk-sh-led">
        {charges.lines.map((line, index) => (
          <div key={line.key} className="dk-sh-lr" style={{ animationDelay: `${index * 40}ms` }}>
            <span>
              <b>{line.label}</b>
              {line.detail && <em>{line.detail}</em>}
            </span>
            <span className="dk-sh-n">{money(line.amount)}</span>
          </div>
        ))}
        <div className="dk-sh-lt">
          <span>{t("Total")}</span>
          <span className="dk-sh-n">{money(charges.total)}</span>
        </div>
      </div>
      {(charges.ratedBy || charges.overridden || charges.locked) && (
        <div className="dk-sh-notes">
          {charges.ratedBy && <span>{charges.ratedBy}</span>}
          {charges.overridden && (
            <span className="dk-warn">
              {charges.overrideReason
                ? t("Overridden: {0}", charges.overrideReason)
                : t("Overridden")}
            </span>
          )}
          {charges.locked && <span className="dk-ink">{t("Rate locked")}</span>}
        </div>
      )}
    </>
  );
}

/**
 * The margin for a reader without the margins field group: the server withholds the
 * shipment's estimate, so the card says so rather than leaving the margin out unexplained.
 */
export function MarginWithheld() {
  const t = useT();
  return (
    <div className="dk-sh-margin dk-unknown">
      <div className="dk-sh-mh">
        <span>{t("Margin")}</span>
        <WithheldValue group="Margins" />
      </div>
    </div>
  );
}

/**
 * The margin against its target: a bar filled to the margin with a tick where
 * the target is, and the profit, cost and rate per mile it comes from.
 */
export function MarginMeter({ margin }: { margin: Margin }) {
  const t = useT();
  const { fill, tick, tone } = marginGauge(margin);
  const percent = margin.percent === null ? "—" : formatPercent(margin.percent);
  return (
    <div className={cn("dk-sh-margin", `dk-${tone}`)}>
      <div className="dk-sh-mh">
        <span>{t("Margin")}</span>
        <b>{percent}</b>
        {margin.target !== null && <em>{t("target {0}", formatPercent(margin.target))}</em>}
      </div>
      <div
        className="dk-sh-meter"
        role="img"
        aria-label={
          margin.target !== null
            ? t("{0} margin against a {1} target", percent, formatPercent(margin.target))
            : t("{0} margin", percent)
        }
      >
        <span className="dk-sh-fill" style={{ width: `${fill * 100}%` }} />
        {tick !== null && <i className="dk-sh-tick" style={{ left: `${tick * 100}%` }} />}
      </div>
      <dl className="dk-sh-mf">
        <div>
          <dt>{t("Profit")}</dt>
          <dd>{money(margin.profit)}</dd>
        </div>
        <div>
          <dt>{t("Estimated cost")}</dt>
          <dd>{money(margin.cost)}</dd>
        </div>
        <div>
          <dt>{t("Rate per mile")}</dt>
          <dd>{margin.ratePerMile !== null ? t("{0}/mi", money(margin.ratePerMile)) : "—"}</dd>
        </div>
      </dl>
      {margin.missingDistance && (
        <p className="dk-sh-note dk-warn">{t("Distance missing; margin is a rough estimate")}</p>
      )}
    </div>
  );
}
