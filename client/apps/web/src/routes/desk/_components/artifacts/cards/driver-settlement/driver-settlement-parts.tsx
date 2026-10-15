import { AlertCircleIcon, AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { minor, signedMinor } from "../../record-card/settlement-values";
import {
  type DisputeCase,
  type FallStep,
  fallScale,
  type SettlementFlag,
} from "./driver-settlement-card";

/**
 * How the statement adds up, as a waterfall: each step a bar from where the
 * running total stood to where it lands (earnings filled, what was taken off
 * hatched), the net pay a full bar from zero in ink, and a shortfall carried to
 * the next settlement hanging below zero in amber, with zero ruled through
 * every lane when it does.
 */
export function PayWaterfall({
  steps,
  currency,
}: {
  steps: readonly FallStep[];
  currency: string;
}) {
  const t = useT();
  const { low, high } = fallScale(steps);
  const span = high - low;
  const at = (value: number) => ((value - low) / span) * 100;
  return (
    <ol className="dk-pay-fall" aria-label={t("How the net pay adds up")}>
      {steps.map((step) => (
        <li key={step.key} className={`dk-${step.sign}`}>
          <span className="dk-what">
            {step.label}
            {step.note && <em>{step.note}</em>}
          </span>
          <span className="dk-lane" aria-hidden>
            {low < 0 && <i className="dk-zero" style={{ left: `${at(0)}%` }} />}
            <i
              className="dk-bar"
              style={{
                left: `${at(step.from)}%`,
                width: `${Math.max(at(step.to) - at(step.from), 0.6)}%`,
              }}
            />
          </span>
          <span className="dk-amt">
            {step.sign === "in"
              ? t("+{0}", minor(step.amount, currency))
              : step.sign === "total"
                ? minor(step.amount, currency)
                : signedMinor(step.amount, currency)}
          </span>
        </li>
      ))}
    </ol>
  );
}

/** The flags the settlement run raised, each a reason to look before approving; a critical one in red. */
export function SettlementFlags({ flags }: { flags: readonly SettlementFlag[] }) {
  return (
    <ul className="dk-pay-flags">
      {flags.map((flag) => {
        const Icon = flag.severity === "critical" ? AlertCircleIcon : AlertTriangleIcon;
        return (
          <li key={flag.key} className={`dk-${flag.severity}`}>
            <Icon className="size-3.5" aria-hidden />
            <span>
              <b>{flag.title}</b>
              {flag.body}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

/** The worker's disputes: an open one as an amber slip in the worker's words, a closed one as how it ended. */
export function SettlementDisputes({ cases }: { cases: readonly DisputeCase[] }) {
  return (
    <ul className="dk-pay-disputes">
      {cases.map((item) => (
        <li key={item.id} className={cn(item.open && "dk-open")}>
          <div className="dk-pay-dispute-h">
            <b>{item.title}</b>
            <span>{item.status}</span>
          </div>
          {item.body && <blockquote>{item.body}</blockquote>}
          <em>{item.meta}</em>
        </li>
      ))}
    </ul>
  );
}
