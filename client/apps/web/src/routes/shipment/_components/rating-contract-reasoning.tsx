import { rateQuoteOutcomeChoices } from "@/lib/choices";
import { rateQuote } from "@/lib/queries/rate-quote";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { RateQuote, RateTraceCandidate } from "@trenova/shared/types/rate";

/**
 * Why the contract that priced the shipment won: the outcome, the rule and any tie-break, what a
 * rate set by hand gave up against the contract and why, every rule that was considered and the
 * reason it lost, and the rating's warnings. It reads the quote the shipment is billed from when
 * it mounts, so its host mounts it only when someone asks.
 */
export function RatingContractReasoning({ shipmentId }: { shipmentId: string }) {
  const t = useT();
  const {
    data: quote,
    isLoading,
    isError,
    refetch,
  } = useQuery({ ...rateQuote.appliedForShipment(shipmentId), staleTime: 60_000 });

  if (isLoading) {
    return (
      <div aria-busy aria-label={t("Reading the rate quote…")} className="flex flex-col gap-1.5">
        <div className="ui-shimmer h-3.5 w-40 rounded-sm" />
        <div className="ui-shimmer h-3 w-full rounded-sm" />
        <div className="ui-shimmer h-3 w-2/3 rounded-sm" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex items-center justify-between gap-2">
        <p className="text-foreground-muted text-xs">
          {t("The contract's reasoning could not be loaded.")}
        </p>
        <Button type="button" variant="ghost" size="xxxs" onClick={() => void refetch()}>
          <span className="text-2xs">{t("Retry")}</span>
        </Button>
      </div>
    );
  }

  if (!quote) {
    return (
      <p className="text-foreground-muted text-xs">
        {t("The quote this shipment was priced from is no longer applied.")}
      </p>
    );
  }
  return <QuoteReasoning quote={quote} />;
}

function QuoteReasoning({ quote }: { quote: RateQuote }) {
  const t = useT();

  const outcome = rateQuoteOutcomeChoices.find((option) => option.value === quote.outcome);
  const trace = quote.trace;
  const winner = trace?.candidates?.find((candidate) => candidate.won);
  const losers = (trace?.candidates ?? []).filter((candidate) => !candidate.won);
  const warnings = trace?.warnings ?? [];

  return (
    <section aria-label={t("Why this contract")} className="flex flex-col gap-3">
      <div className="flex flex-col gap-0.5">
        <div className="flex items-center justify-between gap-2">
          <span
            className={cn(
              "truncate text-xs",
              winner?.agreementName ? "font-medium" : "text-foreground-muted",
            )}
          >
            {winner?.agreementName || t("No contract covered this lane")}
          </span>
          <Badge variant="neutral" appearance="outline" className="shrink-0">
            {outcome ? t(outcome.label) : quote.outcome}
          </Badge>
        </div>
        {winner?.ruleLabel && (
          <span className="text-foreground-muted text-xs">{winner.ruleLabel}</span>
        )}
        {trace?.tieBreak && (
          <span className="text-foreground-muted text-xs">
            {t("Chosen on {0}.", trace.tieBreak)}
          </span>
        )}
      </div>

      {quote.foregoneAmount != null && (
        <p className="text-foreground-muted text-xs">
          {t(
            "This rate was set by hand. The contract would have charged {0}, a difference of {1}. {2}",
            formatCurrency(
              Number(quote.linehaulAmount ?? 0) + Number(quote.foregoneAmount),
              quote.billingCurrency,
            ),
            formatCurrency(Number(quote.foregoneAmount), quote.billingCurrency),
            quote.overrideReason ? t("Reason given: {0}", quote.overrideReason) : "",
          )}
        </p>
      )}

      {losers.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <span className="text-foreground-muted text-xs font-medium">
            {t("Considered but not applied")}
          </span>
          <ul aria-label={t("Considered but not applied")} className="flex flex-col gap-1.5">
            {losers.map((candidate) => (
              <LoserRow key={candidate.ruleId} candidate={candidate} />
            ))}
          </ul>
        </div>
      )}

      {warnings.length > 0 && (
        <ul aria-label={t("Warnings")} className="flex flex-col gap-0.5">
          {warnings.map((warning) => (
            <li key={warning} className="text-foreground-subtle text-2xs">
              {warning}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function LoserRow({ candidate }: { candidate: RateTraceCandidate }) {
  return (
    <li className="flex flex-col gap-0.5">
      <span className="text-xs">{candidate.ruleLabel || candidate.laneKey}</span>
      <span className="text-foreground-subtle text-2xs">{candidate.rejectDetail}</span>
    </li>
  );
}
