import { Button } from "@trenova/shared/components/ui/button";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { humanizeToolName } from "@/components/assistant/proposal-state";
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

/** A price as a ledger: what priced it, every charge with a bar to its running total, and the limits that held. */
export function DeskRateBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const rate = useMemo(() => rateExplanationFrom(artifact), [artifact]);
  const [rejectedOpen, setRejectedOpen] = useState(false);
  const total = Number(rate.totals.total) || 0;

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
            <span>
              {t("Formula template")}
              {rate.pricedBy.expression && (
                <>
                  {" · "}
                  <code>{rate.pricedBy.expression}</code>
                </>
              )}
            </span>
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
        {rate.components.map((component, index) => {
          const running = Number(component.runningTotal) || 0;
          const amount = Number(component.amount) || 0;
          return (
            <div
              key={`${component.label}-${index}`}
              className={cn("dk-ax-lr", amount < 0 && "dk-neg")}
              style={{ animationDelay: `${index * 60}ms` }}
            >
              <span className="dk-ax-ll">
                <b>{component.label}</b>
                <span>{component.basis}</span>
              </span>
              <span className="dk-ax-bar">
                <i
                  style={{
                    width: `${total > 0 ? Math.max(0, Math.min(1, running / total)) * 100 : 0}%`,
                  }}
                />
              </span>
              <span className="dk-ax-num">{money(component.amount, rate.currency)}</span>
            </div>
          );
        })}
        {rate.guardrails.map((guard) => (
          <div key={guard.kind} className="dk-ax-guard">
            <ArtIcon name="check" size={12} stroke={2.4} />
            {humanizeToolName(guard.kind)} {money(guard.bound, rate.currency)} ·{" "}
            {money(guard.result, rate.currency)}
          </div>
        ))}
        <div className="dk-ax-total">
          <span>{t("Total")}</span>
          <b>{money(rate.totals.total, rate.currency)}</b>
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
