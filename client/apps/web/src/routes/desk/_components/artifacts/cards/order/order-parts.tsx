import type { DeskOrderCharge } from "@/lib/graphql/order-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CardFact } from "../../record-card/record-card";
import { money } from "../../record-card/record-card-values";
import { chargeLine, chargeTotals } from "./order-card";

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
