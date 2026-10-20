import type { DeskFuelPurchaseCard } from "@/lib/graphql/fuel-desk-cards";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { cardTime } from "../../record-card/record-card-values";
import {
  iftaBadge,
  iftaLines,
  iftaNote,
  purchasePlace,
  purchaseTitle,
  quarterLabel,
  receiptCard,
  receiptLines,
  totalText,
} from "./fuel-purchase-card";

/** The day a receipt prints: the date in figures, as a pump's slip does. */
function slipDate(seconds: number): string {
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    month: "2-digit",
    day: "2-digit",
    year: "2-digit",
  });
}

/**
 * The purchase drawn as the slip it was rung up on: the vendor and when, each
 * line the purchase records, the total under a rule, and the card and the
 * reference at the foot. A line the card worked out from the others is set
 * apart, and one that does not agree with them is underlined.
 */
export function FuelReceipt({ card }: { card: DeskFuelPurchaseCard }) {
  const t = useT();
  const { purchase } = card;
  const lines = receiptLines(purchase, t);
  const paidWith = receiptCard(purchase, t);
  const reference = purchase.transactionReference?.trim() ?? "";

  return (
    <figure className="dk-fpu-slip" aria-label={t("Receipt as recorded")}>
      <header>
        <b>{purchaseTitle(purchase, t)}</b>
        <span>{purchasePlace(purchase, t)}</span>
        <span>{t("{0} {1}", slipDate(purchase.purchasedAt), cardTime(purchase.purchasedAt))}</span>
      </header>
      <dl>
        {lines.map((line) => (
          <div
            key={line.key}
            className={cn(
              line.mark === "derived" && "dk-fpu-conv",
              line.mark === "off" && "dk-fpu-diff",
              line.mark === "missing" && "dk-fpu-miss",
            )}
          >
            <dt>{line.label}</dt>
            <dd>{line.value}</dd>
          </div>
        ))}
      </dl>
      <div className="dk-fpu-total">
        <span>{t("Total")}</span>
        <b>{totalText(purchase)}</b>
      </div>
      {(paidWith || reference) && (
        <dl className="dk-fpu-tail">
          {paidWith && (
            <div>
              <dt>{t("Card")}</dt>
              <dd>{paidWith}</dd>
            </div>
          )}
          {reference && (
            <div>
              <dt>{t("Ref")}</dt>
              <dd>{reference}</dd>
            </div>
          )}
        </dl>
      )}
    </figure>
  );
}

/**
 * What the purchase does to the quarter's IFTA return: the jurisdiction it
 * counts in, the gallons it adds and how many of them were taxed at the pump,
 * the credit those earn at the quarter's rate, and what the tractor has bought
 * there so far this quarter.
 */
export function FuelIftaPanel({ card }: { card: DeskFuelPurchaseCard }) {
  const t = useT();
  const { purchase, ifta } = card;
  const badge = iftaBadge(card, t);

  return (
    <div className="dk-fpu-ifta">
      <div className="dk-fpu-juris">
        <span className="dk-fpu-code">{purchase.jurisdiction.code}</span>
        <span>
          <b>{purchase.jurisdiction.name}</b>
          <span>{t("IFTA, {0}", quarterLabel(ifta.year, ifta.quarter, t))}</span>
        </span>
        <Badge variant={badge.tone}>{badge.label}</Badge>
      </div>
      <dl>
        {iftaLines(card, t).map((line) => (
          <div key={line.key} className={cn(line.zero && "dk-fpu-zero")}>
            <dt>{line.label}</dt>
            <dd>
              {line.value}
              {line.note && <small className="dk-fu-dim">{line.note}</small>}
            </dd>
          </div>
        ))}
      </dl>
      <p>{iftaNote(card, t)}</p>
    </div>
  );
}
