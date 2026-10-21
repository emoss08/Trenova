import { recordPath } from "@/config/record-links";
import type { DeskTractorPosition } from "@/lib/graphql/tractor-card";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatPercent, metersToMiles } from "@trenova/shared/lib/utils";
import { CardAvatar, CardRecordRow, type CardFact } from "../../record-card/record-card";
import { compassPoint, engineText, type TractorPerson } from "./tractor-card";

/** Below this much fuel the gauge reads low. */
const LOW_FUEL = 0.25;

/**
 * The tractor's dashboard at its last report: how fast it was going and which
 * way, how much fuel it had on a ring, whether the engine was running, and the
 * miles on its clock. Each reading says "not reported" rather than a zero when
 * telematics left it out.
 */
export function TractorDash({ position }: { position: DeskTractorPosition }) {
  const t = useT();
  const speed = Math.round(position.speedMph);
  const fuel =
    position.fuelPercent === null || position.fuelPercent === undefined
      ? null
      : Math.min(1, Math.max(0, position.fuelPercent / 100));
  const engine = position.engineState?.trim() || "";
  return (
    <div className="dk-tr-dash">
      <div className="dk-tr-dial">
        <b>{speed}</b>
        <span>{t("mph")}</span>
      </div>
      <div className="dk-tr-dial">
        <svg
          className="dk-tr-arrow"
          viewBox="0 0 24 24"
          width={22}
          height={22}
          style={{ transform: `rotate(${position.headingDegrees}deg)` }}
          aria-hidden
        >
          <path d="M12 3l6 17-6-4-6 4z" />
        </svg>
        <span>{t("Heading {0}", compassPoint(position.headingDegrees))}</span>
      </div>
      <div className="dk-tr-dial">
        {fuel === null ? (
          <b className="dk-tr-none">—</b>
        ) : (
          <RingGauge
            value={fuel}
            size={40}
            strokeWidth={4}
            tone={fuel < LOW_FUEL ? "warning" : "muted"}
            aria-label={t("Fuel")}
          >
            <span className="dk-tr-fuel">{formatPercent(position.fuelPercent ?? 0, 0)}</span>
          </RingGauge>
        )}
        <span>{fuel === null ? t("Fuel not reported") : t("Fuel")}</span>
      </div>
      <div className="dk-tr-dial">
        <i className={cn("dk-tr-engine", engine && `dk-${engine.toLowerCase()}`)} aria-hidden />
        <span>{engineText(engine, t)}</span>
      </div>
      <div className="dk-tr-dial dk-tr-odo">
        <b>
          {position.odometerMeters
            ? formatNumber(Math.round(metersToMiles(position.odometerMeters)))
            : "—"}
        </b>
        <span>{t("Odometer, mi")}</span>
      </div>
    </div>
  );
}

/** Who drives the tractor, each in a seat with their initials, opened in the card's place. */
export function DriverSeats({
  people,
  onOpen,
}: {
  people: readonly TractorPerson[];
  onOpen: ((workerId: string) => void) | undefined;
}) {
  return (
    <div className="dk-tr-seats">
      {people.map((person) => (
        <CardRecordRow
          key={person.key}
          className="dk-tr-seat"
          kind="worker" to={recordPath("worker", person.id)}
          onOpen={onOpen && (() => onOpen(person.id))}
        >
          <CardAvatar name={person.name} />
          <span className="dk-rec-main">
            <b>{person.name}</b>
            <span className="dk-rec-sub">{person.role}</span>
          </span>
        </CardRecordRow>
      ))}
    </div>
  );
}

/** The tractor's own facts as a spec sheet: each label run out to its value on a dotted leader. */
export function SpecSheet({ facts }: { facts: readonly CardFact[] }) {
  return (
    <dl className="dk-tr-spec">
      {facts.map((fact) => (
        <div key={fact.key}>
          <dt>{fact.label}</dt>
          <i aria-hidden />
          <dd>{fact.value}</dd>
        </div>
      ))}
    </dl>
  );
}
