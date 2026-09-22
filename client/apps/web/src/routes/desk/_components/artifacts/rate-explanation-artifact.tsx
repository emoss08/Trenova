import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { toneVar } from "@/components/kpi/tone";
import { TriangleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import { rateExplanationFrom } from "./artifact-payloads";

/**
 * Why a shipment costs what it does, as a ledger.
 *
 * A price read aloud is a wall of figures nobody can check. Laid out beside
 * its arithmetic and a running total, it is the thing a person puts next to
 * the invoice line a customer is disputing — every charge with the basis the
 * engine recorded ("1,240.0 mi @ $2.15/mi"), the limits that changed the
 * number, and what priced it in the first place.
 */
export function RateExplanationArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const rate = useMemo(() => rateExplanationFrom(artifact), [artifact]);

  return (
    <div className="scrollbar-overlay flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-3">
      {rate.winner && (
        <div className="space-y-0.5">
          <p className="text-xs font-medium">{t("Priced under")}</p>
          <p className="text-sm">
            {rate.winner.ruleLabel !== "" ? rate.winner.ruleLabel : rate.winner.agreementName}
          </p>
          <p className="text-muted-foreground text-xs">
            {rate.winner.agreementName !== "" && rate.winner.ruleLabel !== ""
              ? rate.winner.agreementName
              : rate.winner.agreementCode}
            {rate.tieBreak !== "" ? ` · ${t("chosen on {0}", rate.tieBreak)}` : ""}
          </p>
        </div>
      )}

      <table className="w-full text-xs">
        <tbody>
          {rate.components.map((component, index) => (
            <tr key={`${component.label}-${index}`} className="border-border/60 border-b">
              <td className="py-1.5 pr-2">
                <span className="block">{component.label}</span>
                {component.basis !== "" && (
                  <span className="text-muted-foreground block">{component.basis}</span>
                )}
              </td>
              <td className="py-1.5 pr-2 text-right tabular-nums">{component.amount}</td>
              <td className="text-muted-foreground py-1.5 text-right tabular-nums">
                {component.runningTotal}
              </td>
            </tr>
          ))}
          <tr>
            <td className="py-1.5 pr-2 font-medium">{t("Total")}</td>
            <td colSpan={2} className="py-1.5 text-right font-medium tabular-nums">
              {rate.totals.total}
              {rate.currency !== "" ? ` ${rate.currency}` : ""}
            </td>
          </tr>
        </tbody>
      </table>

      {/* Only the limits that actually bit are here, so each one is the reason
          the total is not what the charges add up to. */}
      {rate.guardrails.length > 0 && (
        <div className="space-y-1">
          <p className="text-xs font-medium">{t("Limits applied")}</p>
          {rate.guardrails.map((guardrail, index) => (
            <p key={`${guardrail.kind}-${index}`} className="text-muted-foreground text-xs">
              {t("{0}: {1} became {2}", guardrail.kind, guardrail.raw, guardrail.result)}
            </p>
          ))}
        </div>
      )}

      {/* "No rate applied" and "a rate applied, but not the one you expected"
          are different problems, and this is what tells them apart. */}
      {rate.rejected.length > 0 && (
        <div className="space-y-1">
          <p className="text-xs font-medium">{t("Passed over")}</p>
          {rate.rejected.map((rejected, index) => (
            <p key={`${rejected.agreementCode}-${index}`} className="text-xs">
              <span className="block">
                {rejected.ruleLabel !== "" ? rejected.ruleLabel : rejected.agreementCode}
              </span>
              <span className="text-muted-foreground block">
                {rejected.reason}
                {rejected.detail !== "" ? ` — ${rejected.detail}` : ""}
              </span>
            </p>
          ))}
        </div>
      )}

      {rate.warnings.map((warning) => (
        <p
          key={warning}
          className={cn("flex items-start gap-1.5 text-xs")}
          style={{ color: toneVar("warning") }}
        >
          <TriangleAlertIcon className="mt-px size-3 shrink-0" />
          {warning}
        </p>
      ))}
    </div>
  );
}
