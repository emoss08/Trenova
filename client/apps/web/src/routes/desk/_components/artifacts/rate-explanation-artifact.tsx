import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { TriangleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import { rateExplanationFrom } from "./artifact-payloads";
import { ArtifactScroll, ArtifactSection } from "./artifact-section";

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
    <ArtifactScroll>
      {rate.winner && (
        <ArtifactSection title={t("Priced under")}>
          <p className="text-sm">
            {rate.winner.ruleLabel !== "" ? rate.winner.ruleLabel : rate.winner.agreementName}
          </p>
          <p className="text-muted-foreground text-xs">
            {rate.winner.agreementName !== "" && rate.winner.ruleLabel !== ""
              ? rate.winner.agreementName
              : rate.winner.agreementCode}
            {rate.tieBreak !== "" ? ` · ${t("chosen on {0}", rate.tieBreak)}` : ""}
          </p>
        </ArtifactSection>
      )}

      <ArtifactSection
        title={t("Charges")}
        hint={rate.currency !== "" ? rate.currency : undefined}
        inset={false}
      >
        <table className="w-full text-xs">
          <tbody className="divide-border-subtle divide-y">
            {rate.components.map((component, index) => (
              <tr key={`${component.label}-${index}`}>
                <td className="py-1.5 pr-2 pl-3">
                  <span className="block">{component.label}</span>
                  {component.basis !== "" && (
                    <span className="text-muted-foreground block">{component.basis}</span>
                  )}
                </td>
                <td className="py-1.5 pr-2 text-right tabular-nums">{component.amount}</td>
                <td className="text-muted-foreground py-1.5 pr-3 text-right tabular-nums">
                  {component.runningTotal}
                </td>
              </tr>
            ))}
            <tr className="bg-sunken/60">
              <td className="py-1.5 pr-2 pl-3 font-medium">{t("Total")}</td>
              <td colSpan={2} className="py-1.5 pr-3 text-right font-medium tabular-nums">
                {rate.totals.total}
                {rate.currency !== "" ? ` ${rate.currency}` : ""}
              </td>
            </tr>
          </tbody>
        </table>
      </ArtifactSection>

      {/* Only the limits that actually bit are here, so each one is the reason
          the total is not what the charges add up to. */}
      {rate.guardrails.length > 0 && (
        <ArtifactSection title={t("Limits applied")} inset={false}>
          <ul className="divide-border-subtle flex flex-col divide-y">
            {rate.guardrails.map((guardrail, index) => (
              <li
                key={`${guardrail.kind}-${index}`}
                className="text-muted-foreground px-3 py-2 text-xs"
              >
                {t("{0}: {1} became {2}", guardrail.kind, guardrail.raw, guardrail.result)}
              </li>
            ))}
          </ul>
        </ArtifactSection>
      )}

      {/* "No rate applied" and "a rate applied, but not the one you expected"
          are different problems, and this is what tells them apart. */}
      {rate.rejected.length > 0 && (
        <ArtifactSection title={t("Passed over")} inset={false}>
          <ul className="divide-border-subtle flex flex-col divide-y">
            {rate.rejected.map((rejected, index) => (
              <li key={`${rejected.agreementCode}-${index}`} className="px-3 py-2 text-xs">
                <span className="block">
                  {rejected.ruleLabel !== "" ? rejected.ruleLabel : rejected.agreementCode}
                </span>
                <span className="text-muted-foreground block">
                  {rejected.reason}
                  {rejected.detail !== "" ? ` — ${rejected.detail}` : ""}
                </span>
              </li>
            ))}
          </ul>
        </ArtifactSection>
      )}

      {rate.warnings.length > 0 && (
        <ul className="flex flex-col gap-1.5">
          {rate.warnings.map((warning) => (
            <li key={warning} className="text-warning flex items-start gap-1.5 text-xs">
              <TriangleAlertIcon aria-hidden className="mt-px size-3 shrink-0" />
              {warning}
            </li>
          ))}
        </ul>
      )}
    </ArtifactScroll>
  );
}
