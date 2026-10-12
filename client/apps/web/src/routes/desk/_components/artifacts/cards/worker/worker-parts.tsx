import { hosClockTone } from "@/components/hos/hos-clock-gauges";
import { Mail01Icon, Phone01Icon } from "@trenova/shared/components/icons";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { worstFirst, type DueCheck } from "../../record-card/card-due";
import {
  CardIcon,
  CardSection,
  needsLook,
  type CardCheckState,
  type CardFact,
} from "../../record-card/record-card";
import type { HosClock } from "./worker-card";

/** How big the clocks' rings are drawn, and how thick, as the worker's own page draws them. */
const RING_SIZE = 88;
const RING_STROKE = 6;

/**
 * The driver's four clocks as rings, each filled with what is left of its
 * limit and the time left written inside, coloured as the worker's page
 * colours them: short under two hours, out under one.
 */
export function HosRings({ clocks }: { clocks: readonly HosClock[] }) {
  const t = useT();
  return (
    <div className="dk-wk-rings">
      {clocks.map((clock) => (
        <div key={clock.key} className={cn("dk-wk-ring", `dk-${clock.state}`)}>
          <RingGauge
            value={clock.share}
            size={RING_SIZE}
            strokeWidth={RING_STROKE}
            tone={hosClockTone(clock.remainingMs)}
            aria-label={t("{0} time remaining", clock.label)}
          >
            <span className="dk-wk-ring-v">{formatClockDurationMs(clock.remainingMs)}</span>
          </RingGauge>
          <b>{clock.label}</b>
          <span>{t("of {0}", formatClockDurationMs(clock.limitMs))}</span>
        </div>
      ))}
    </div>
  );
}

const STATE_ICON: Record<CardCheckState, "check" | "warn" | "x" | "info"> = {
  ok: "check",
  warn: "warn",
  bad: "x",
  info: "info",
};

/**
 * The worker's credentials as a wallet: a card each, worst first, coloured by
 * whether it holds, falls due soon or has lapsed, with how many need a look
 * beside the title.
 */
export function CredentialWallet({
  title,
  checks,
}: {
  title: string;
  checks: readonly DueCheck[];
}) {
  const t = useT();
  const open = checks.filter((check) => needsLook(check.state)).length;
  return (
    <CardSection
      title={title}
      aside={
        open > 0 ? (
          <span className="dk-warn">
            {t("{0, plural, one {# needs a look} other {# need a look}}", open)}
          </span>
        ) : (
          <span className="dk-ok">{t("Nothing due")}</span>
        )
      }
    >
      <ul className="dk-wk-wallet">
        {worstFirst(checks).map((check) => (
          <li key={check.key} className={`dk-${check.state}`}>
            <span className="dk-wk-wallet-i" aria-hidden>
              <CardIcon name={STATE_ICON[check.state]} size={11} stroke={2.6} />
            </span>
            <b>{check.title}</b>
            <span>{check.detail}</span>
          </li>
        ))}
      </ul>
    </CardSection>
  );
}

/**
 * How to reach the worker, as ways to call or write to them, and what their
 * employment record says, as a list read down.
 */
export function WorkerReach({
  phone,
  email,
  facts,
}: {
  phone: string;
  email: string;
  facts: readonly CardFact[];
}) {
  const t = useT();
  return (
    <div className="dk-wk-reach">
      <div className="dk-wk-ways">
        {phone ? (
          <a className="dk-wk-way" href={`tel:${phone.replace(/[^\d+]/gu, "")}`}>
            <Phone01Icon className="size-3.5" aria-hidden />
            <span>
              <em>{t("Call")}</em>
              <b>{phone}</b>
            </span>
          </a>
        ) : (
          <span className="dk-wk-way dk-off">
            <Phone01Icon className="size-3.5" aria-hidden />
            <span>
              <em>{t("Call")}</em>
              <b>{t("No phone on file")}</b>
            </span>
          </span>
        )}
        {email ? (
          <a className="dk-wk-way" href={`mailto:${email}`}>
            <Mail01Icon className="size-3.5" aria-hidden />
            <span>
              <em>{t("Email")}</em>
              <b>{email}</b>
            </span>
          </a>
        ) : (
          <span className="dk-wk-way dk-off">
            <Mail01Icon className="size-3.5" aria-hidden />
            <span>
              <em>{t("Email")}</em>
              <b>{t("No email on file")}</b>
            </span>
          </span>
        )}
      </div>
      <dl className="dk-wk-record">
        {facts.map((fact) => (
          <div key={fact.key}>
            <dt>{fact.label}</dt>
            <dd>{fact.value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
