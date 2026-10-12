import type { DeskOrderCharge } from "@/lib/graphql/order-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CardIcon, type CardFact } from "../../record-card/record-card";
import { cardMoment, money } from "../../record-card/record-card-values";
import { chargeLine, chargeTotals, type LifecycleStep } from "./order-card";

/**
 * The order's way from draft to closed as one track of segments, each naming
 * its step: the steps behind it ticked, the one it stands at filled, the ones
 * ahead left open, with which step of how many it is and when it last
 * changed. A canceled order leaves the whole track off.
 */
export function LifecycleTrack({
  steps,
  canceled,
  changedAt,
}: {
  steps: readonly LifecycleStep[];
  canceled: boolean;
  changedAt: number;
}) {
  const t = useT();
  const at = steps.findIndex((step) => step.state === "now");
  return (
    <div className={cn("dk-or-track", canceled && "dk-off")}>
      <ol aria-label={t("Order lifecycle")}>
        {steps.map((step) => (
          <li
            key={step.key}
            className={`dk-${step.state}`}
            aria-current={step.state === "now" ? "step" : undefined}
          >
            {step.state === "done" && <CardIcon name="check" size={11} stroke={2.8} />}
            <span>{step.label}</span>
          </li>
        ))}
      </ol>
      <p>
        {canceled
          ? t("Canceled · last changed {0}", cardMoment(changedAt))
          : t("Step {0} of {1} · last changed {2}", at + 1, steps.length, cardMoment(changedAt))}
      </p>
    </div>
  );
}

/**
 * The order's charges as a ledger: a bar of how much has been invoiced against
 * what is still to invoice, then each charge on a line with its amount after a
 * dotted leader, and their total under a rule.
 */
export function ChargeLedger({
  charges,
  currency,
}: {
  charges: readonly DeskOrderCharge[];
  currency: string;
}) {
  const t = useT();
  const totals = chargeTotals(charges);
  const invoiced = totals.total > 0 ? Math.round((totals.invoiced / totals.total) * 100) : 0;
  return (
    <div className="dk-or-ledger">
      <div className="dk-or-billed">
        <div
          className="dk-or-billbar"
          role="meter"
          aria-label={t("Invoiced")}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={invoiced}
        >
          <i style={{ width: `${invoiced}%` }} />
        </div>
        <p>
          <span className="dk-or-key dk-done">
            {t("Invoiced {0}", money(totals.invoiced / 100, currency))}
          </span>
          <span className="dk-or-key">
            {t("Still to invoice {0}", money(totals.open / 100, currency))}
          </span>
        </p>
      </div>
      <ul>
        {charges.map((charge) => (
          <li key={charge.id} className={cn(charge.invoicedAt && "dk-done")}>
            <span className="dk-or-line">
              <b>{charge.description}</b>
              <i aria-hidden />
              <span className="dk-or-amt">{money(charge.amount, currency)}</span>
            </span>
            <span className="dk-rec-sub">{chargeLine(charge, t)}</span>
          </li>
        ))}
      </ul>
      <div className="dk-or-total">
        <span>{t("Charges")}</span>
        <b>{money(totals.total / 100, currency)}</b>
      </div>
    </div>
  );
}

/** The order's numbers as references to quote, and what else is on its record, on one strip. */
export function OrderReferences({ facts }: { facts: readonly CardFact[] }) {
  return (
    <dl className="dk-or-refs">
      {facts.map((fact) => (
        <div key={fact.key}>
          <dt>{fact.label}</dt>
          <dd>{fact.value}</dd>
        </div>
      ))}
    </dl>
  );
}
