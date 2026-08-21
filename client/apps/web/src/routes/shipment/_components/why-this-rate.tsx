import { rateQuoteOutcomeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { RateQuote, RateTraceCandidate } from "@trenova/shared/types/rate";
import { InfoIcon } from "lucide-react";

type WhyThisRateProps = {
  shipmentId?: string;
};

/**
 * The answer to "why did I get this rate", which is the complaint every
 * carrier and broker has about the system they are leaving.
 *
 * It shows the contract and lane that won, what each term contributed, and —
 * the part that actually settles arguments — every rate that was considered and
 * the reason it lost.
 */
export function WhyThisRate({ shipmentId }: WhyThisRateProps) {
  const { data: quote, isLoading } = useQuery({
    ...queries.rateQuote.appliedForShipment(shipmentId ?? ""),
    enabled: Boolean(shipmentId),
  });

  if (!shipmentId || (!isLoading && !quote)) {
    return null;
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button type="button" variant="ghost" size="xxxs">
            <InfoIcon className="size-3" />
            <span className="text-2xs">Why this rate</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="w-104 p-0">
        {quote ? <QuoteExplanation quote={quote} /> : <LoadingState />}
      </PopoverContent>
    </Popover>
  );
}

function LoadingState() {
  return <p className="text-2xs text-muted-foreground p-3">Reading the rate quote…</p>;
}

function QuoteExplanation({ quote }: { quote: RateQuote }) {
  const outcome = rateQuoteOutcomeChoices.find((option) => option.value === quote.outcome);
  const trace = quote.trace;
  const winner = trace?.candidates?.find((candidate) => candidate.won);
  const losers = (trace?.candidates ?? []).filter((candidate) => !candidate.won);

  return (
    <div className="flex max-h-[32rem] flex-col overflow-y-auto">
      <div className="border-b p-3">
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-medium">
            {winner?.agreementName || "No contract covered this lane"}
          </span>
          <Badge variant="outline" className="text-[10px]">
            {outcome?.label ?? quote.outcome}
          </Badge>
        </div>
        {winner?.ruleLabel && (
          <p className="text-2xs text-muted-foreground mt-0.5">{winner.ruleLabel}</p>
        )}
        {trace?.tieBreak && (
          <p className="text-2xs text-muted-foreground mt-1">Chosen on {trace.tieBreak}.</p>
        )}
      </div>

      {(trace?.components?.length ?? 0) > 0 && (
        <div className="border-b p-3">
          <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
            What made up the rate
          </p>
          <div className="space-y-1.5">
            {trace?.components?.map((component) => (
              <div
                key={`${component.sequence}-${component.label}`}
                className="flex items-baseline justify-between gap-3"
              >
                <div className="flex flex-col">
                  <span className="text-xs">{component.label}</span>
                  {component.basis && (
                    <span className="text-2xs text-muted-foreground">{component.basis}</span>
                  )}
                </div>
                <span className="text-xs tabular-nums">
                  {formatCurrency(Number(component.amount ?? 0))}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {(trace?.guardrails?.length ?? 0) > 0 && (
        <div className="border-b p-3">
          <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
            Guardrails
          </p>
          {trace?.guardrails?.map((guardrail) => (
            <p key={guardrail.kind} className="text-2xs text-muted-foreground">
              {guardrail.label}
              {guardrail.applied
                ? ` applied — ${formatCurrency(Number(guardrail.rawAmount ?? 0))} became ${formatCurrency(Number(guardrail.amount ?? 0))}.`
                : " did not apply."}
            </p>
          ))}
        </div>
      )}

      {quote.foregoneAmount != null && (
        <div className="border-b p-3">
          <p className="text-2xs text-muted-foreground">
            This rate was set by hand. The contract would have charged{" "}
            {formatCurrency(Number(quote.linehaulAmount ?? 0) + Number(quote.foregoneAmount))}, a
            difference of {formatCurrency(Number(quote.foregoneAmount))}.
            {quote.overrideReason ? ` Reason given: ${quote.overrideReason}` : ""}
          </p>
        </div>
      )}

      {losers.length > 0 && (
        <div className="p-3">
          <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
            Considered but not applied
          </p>
          <div className="space-y-1.5">
            {losers.map((candidate) => (
              <LoserRow key={candidate.ruleId} candidate={candidate} />
            ))}
          </div>
        </div>
      )}

      {(trace?.warnings?.length ?? 0) > 0 && (
        <div className="bg-muted/40 border-t p-3">
          {trace?.warnings?.map((warning) => (
            <p key={warning} className="text-2xs text-muted-foreground">
              {warning}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}

function LoserRow({ candidate }: { candidate: RateTraceCandidate }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs">{candidate.ruleLabel || candidate.laneKey}</span>
      <span className="text-2xs text-muted-foreground">{candidate.rejectDetail}</span>
    </div>
  );
}
