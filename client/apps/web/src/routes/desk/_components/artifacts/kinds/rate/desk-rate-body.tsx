import { Button } from "@trenova/shared/components/ui/button";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { tariffLineLabel } from "@/components/tariff-bill/bill-view";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import type { StoredTariffBill } from "@trenova/shared/types/tariff-form";
import { useMemo, useState } from "react";
import { rateExplanationFrom } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";

function money(amount: string, currency: string): string {
  const figure = Number(amount);
  if (amount === "" || !Number.isFinite(figure)) {
    return amount;
  }
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}

type LedgerRow = {
  key: string;
  label: string;
  code: string | null;
  basis: string;
  amountText: string;
  amount: number;
  running: number;
};

/**
 * The bill's lines as ledger rows, each with its running total for the bar: a charge under the
 * tariff's own label, an adjustment (minimum, maximum, rounding) in the reader's words.
 */
function ledgerRows(bill: StoredTariffBill | null, t: TranslateFn, currency: string): LedgerRow[] {
  let running = 0;
  return (bill?.lines ?? []).map((line, index) => {
    const amount = Number(line.amount.$q) || 0;
    running += amount;
    return {
      key: `${line.kind}-${line.label ?? ""}-${index}`,
      label: tariffLineLabel(line, t),
      code: line.code,
      basis:
        line.quantity !== null && line.rate !== null
          ? t(
              "{0} × {1}",
              formatTariffValue(line.quantity, t, { currency }),
              formatTariffValue(line.rate, t, { currency }),
            )
          : "",
      amountText: line.amount.$q,
      amount,
      running,
    };
  });
}

/** A price as a ledger: what priced it, every line of its bill with a bar to its running total, and its notes. */
export function DeskRateBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const rate = useMemo(() => rateExplanationFrom(artifact), [artifact]);
  const [rejectedOpen, setRejectedOpen] = useState(false);
  const total = Number(rate.bill?.total.$q ?? rate.totals.total) || 0;

  return (
    <div className="dk-ax-pad dk-ax-rate">
      {rate.winner && (
        <div className="dk-ax-win">
          <span className="dk-ax-wck">
            <ArtIcon name="check" size={12} stroke={2.6} />
          </span>
          <div>
            <b>{rate.winner.agreementName || rate.winner.agreementCode}</b>
            <span>
              {rate.winner.ruleLabel}
              {rate.winner.agreementCode && (
                <>
                  {" · "}
                  <code>{rate.winner.agreementCode}</code>
                </>
              )}
            </span>
          </div>
        </div>
      )}
      {!rate.winner && rate.pricedBy && (
        <div className="dk-ax-win">
          <span className="dk-ax-wck">
            <ArtIcon name="check" size={12} stroke={2.6} />
          </span>
          <div>
            <b>{rate.pricedBy.method || t("Entered by hand")}</b>
            <span>{rate.pricedBy.method ? t("Tariff") : t("Manual rate")}</span>
          </div>
        </div>
      )}
      {!rate.winner && rate.pricedBy?.explanation && (
        <p className="dk-ax-note">{rate.pricedBy.explanation}</p>
      )}
      {rate.pricedBy?.override && (
        <p className="dk-ax-note">
          {t("Overridden to {0}", money(rate.pricedBy.override, rate.currency))}
          {rate.pricedBy.overrideReason ? ` · ${rate.pricedBy.overrideReason}` : ""}
        </p>
      )}
      {rate.tieBreak !== "" && <p className="dk-ax-note">{rate.tieBreak}</p>}
      <div className="dk-ax-ledger">
        {ledgerRows(rate.bill, t, rate.currency).map((row, index) => (
          <div
            key={row.key}
            className={cn("dk-ax-lr", row.amount < 0 && "dk-neg")}
            style={{ animationDelay: `${index * 60}ms` }}
          >
            <span className="dk-ax-ll">
              <b>
                {row.label}
                {row.code && (
                  <>
                    {" "}
                    <code>{row.code}</code>
                  </>
                )}
              </b>
              <span>{row.basis}</span>
            </span>
            <span className="dk-ax-bar">
              <i
                style={{
                  width: `${total > 0 ? Math.max(0, Math.min(1, row.running / total)) * 100 : 0}%`,
                }}
              />
            </span>
            <span className="dk-ax-num">{money(row.amountText, rate.currency)}</span>
          </div>
        ))}
        {(rate.bill?.notes ?? []).map((note) => (
          <div key={note.text} className="dk-ax-guard">
            <ArtIcon name="check" size={12} stroke={2.4} />
            {note.text}
          </div>
        ))}
        <div className="dk-ax-total">
          <span>{t("Total")}</span>
          <b>{money(rate.bill?.total.$q ?? rate.totals.total, rate.currency)}</b>
        </div>
      </div>
      {rate.warnings.map((warning) => (
        <div key={warning} className="dk-ax-warn">
          <ArtIcon name="warn" size={13} />
          {warning}
        </div>
      ))}
      {rate.rejected.length > 0 && (
        <>
          <Button
            variant="bare"
            size="bare"
            className="mt-3 gap-1.5 text-sm text-dsk-subtle hover:text-dsk-fg"
            onClick={() => setRejectedOpen((value) => !value)}
          >
            <ArtIcon name={rejectedOpen ? "up" : "down"} size={11} stroke={2.2} />
            {t(
              "{0, plural, one {# agreement didn't apply} other {# agreements didn't apply}}",
              rate.rejected.length,
            )}
          </Button>
          {rejectedOpen && (
            <div className="dk-ax-rej">
              {rate.rejected.map((entry) => (
                <div key={entry.agreementCode + entry.ruleLabel}>
                  <code>{entry.agreementCode}</code>
                  <span>{entry.ruleLabel}</span>
                  <em>{entry.detail || entry.reason}</em>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
