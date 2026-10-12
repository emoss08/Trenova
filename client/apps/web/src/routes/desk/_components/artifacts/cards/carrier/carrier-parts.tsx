import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DueCheck } from "../../record-card/card-due";
import {
  CardAvatar,
  CardIcon,
  type CardCheckState,
  type CardStat,
} from "../../record-card/record-card";
import type { BasicMeter, CoverageTile } from "./carrier-card";

const STATE_ICON: Record<CardCheckState, "check" | "warn" | "x" | "info"> = {
  ok: "check",
  warn: "warn",
  bad: "x",
  info: "info",
};

function percentOf(share: number): number {
  return Math.round(Math.min(1, Math.max(0, share)) * 100);
}

/**
 * Whether the carrier can be tendered a load, said as a verdict across the
 * card in the colour of its answer, with what stands in the way listed inside
 * it, worst first.
 */
export function VerdictBanner({
  verdict,
  checks,
}: {
  verdict: { label: string; state: CardCheckState };
  checks: readonly DueCheck[];
}) {
  const t = useT();
  return (
    <div className={cn("dk-ca-verdict", `dk-${verdict.state}`)}>
      <div className="dk-ca-verdict-h">
        <span className="dk-ca-verdict-i" aria-hidden>
          <CardIcon name={STATE_ICON[verdict.state]} size={16} stroke={2.6} />
        </span>
        <b>{verdict.label}</b>
        <span>
          {checks.length > 0
            ? t("{0, plural, one {# finding} other {# findings}}", checks.length)
            : t("Nothing stands in the way")}
        </span>
      </div>
      {checks.length > 0 && (
        <ul>
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
      )}
    </div>
  );
}

/** The carrier's insurance as coverage tiles: each policy's cover large, its insurer and its term. */
export function CoverageTiles({ tiles }: { tiles: readonly CoverageTile[] }) {
  const t = useT();
  return (
    <ul className="dk-ca-cover">
      {tiles.map((tile) => (
        <li key={tile.key} className={`dk-${tile.state}`}>
          <span className="dk-ca-cover-k">{tile.title}</span>
          <b>{tile.amount}</b>
          <span className="dk-ca-cover-by">{tile.insurer || t("No insurer on file")}</span>
          <span className="dk-ca-cover-to">
            {tile.until}
            {tile.unverified && <em>{t("Not verified")}</em>}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** The carrier's operating authority and safety rating, a pill each in the colour of how it stands. */
export function AuthorityPills({ stats }: { stats: readonly CardStat[] }) {
  return (
    <dl className="dk-ca-auth">
      {stats.map((stat) => (
        <div key={stat.key} className={cn(stat.state && `dk-${stat.state}`)}>
          <dt>{stat.label}</dt>
          <dd>{stat.value}</dd>
        </div>
      ))}
    </dl>
  );
}

/**
 * The CSA BASICs as a column each, risen to its percentile against a dashed
 * line at the threshold that sets an alert, coloured by how close it runs.
 * A BASIC with too little data to rank stands empty and says so.
 */
export function BasicChart({ meters }: { meters: readonly BasicMeter[] }) {
  const t = useT();
  return (
    <div className="dk-ca-chart" role="list" aria-label={t("CSA BASICs")}>
      {meters.map((meter) => (
        <div key={meter.key} role="listitem" className={cn("dk-ca-col", `dk-${meter.state}`)}>
          <div
            className="dk-ca-col-bar"
            role="meter"
            aria-label={meter.label}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={percentOf(meter.share)}
            aria-valuetext={meter.reading}
          >
            <i style={{ height: `${percentOf(meter.share)}%` }} />
            {meter.thresholdShare !== null && (
              <span style={{ bottom: `${percentOf(meter.thresholdShare)}%` }} />
            )}
          </div>
          <b aria-hidden>{meter.reading}</b>
          <span aria-hidden>{meter.label}</span>
        </div>
      ))}
    </div>
  );
}

/** One person to call at the carrier, with their initials, how to reach them and their role. */
export function ContactCard({ name, line, role }: { name: string; line: string; role: string }) {
  return (
    <li className="dk-ca-person">
      <CardAvatar name={name} />
      <span className="dk-rec-main">
        <b>{name}</b>
        <span className="dk-rec-sub">{line}</span>
      </span>
      {role && <span className="dk-ca-role">{role}</span>}
    </li>
  );
}
