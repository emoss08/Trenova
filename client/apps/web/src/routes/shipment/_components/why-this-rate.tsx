import { useT } from "@trenova/shared/i18n/use-t";
import { rateGuardrailKindChoices, rateQuoteOutcomeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { RateQuote, RateTraceCandidate, RateTraceGuardrail } from "@trenova/shared/types/rate";
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
  const t = useT();

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
            <span className="text-2xs">{t("Why this rate")}</span>
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
  const t = useT();

  return <p className="text-2xs text-muted-foreground p-3">{t("Reading the rate quote…")}</p>;
}

function QuoteExplanation({ quote }: { quote: RateQuote }) {
  const t = useT();

  const outcome = rateQuoteOutcomeChoices.find((option) => option.value === quote.outcome);
  const trace = quote.trace;
  const winner = trace?.candidates?.find((candidate) => candidate.won);
  const losers = (trace?.candidates ?? []).filter((candidate) => !candidate.won);

  return (
    <div className="flex max-h-128 flex-col overflow-y-auto">
      <div className="border-b p-3">
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-medium">
            {winner?.agreementName || t("No contract covered this lane")}
          </span>
          <Badge variant="outline" className="text-[10px]">
            {outcome ? t(outcome.label) : quote.outcome}
          </Badge>
        </div>
        {winner?.ruleLabel && (
          <p className="text-2xs text-muted-foreground mt-0.5">{winner.ruleLabel}</p>
        )}
        {trace?.tieBreak && (
          <p className="text-2xs text-muted-foreground mt-1">
            {t("Chosen on {0}.", trace.tieBreak)}
          </p>
        )}
      </div>

      {(trace?.components?.length ?? 0) > 0 && (
        <div className="border-b p-3">
          <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
            {t("What made up the rate")}
          </p>
          <div className="space-y-1.5">
            {trace?.components?.map((component) => (
              <div
                key={`${component.sequence}-${component.label}`}
                className="flex items-baseline justify-between gap-3"
              >
                <div className="flex flex-col">
                  <span className="text-xs">{t(component.label)}</span>
                  {component.basis && (
                    <span className="text-2xs text-muted-foreground">{component.basis}</span>
                  )}
                </div>
                <span className="text-xs tabular-nums">
                  {formatCurrency(Number(component.amount ?? 0), quote.currency)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {(trace?.guardrails?.length ?? 0) > 0 && (
        <div className="border-b p-3">
          <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
            {t("Guardrails")}
          </p>
          {trace?.guardrails?.map((guardrail, index) => (
            <GuardrailRow
              key={`${guardrail.kind}-${index}`}
              guardrail={guardrail}
              currency={quote.currency}
            />
          ))}
        </div>
      )}

      {quote.foregoneAmount != null && (
        <div className="border-b p-3">
          <p className="text-2xs text-muted-foreground">
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
        </div>
      )}

      {losers.length > 0 && (
        <div className="p-3">
          <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
            {t("Considered but not applied")}
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

function GuardrailRow({
  guardrail,
  currency,
}: {
  guardrail: RateTraceGuardrail;
  currency: string;
}) {
  const t = useT();

  const kind = rateGuardrailKindChoices.find((option) => option.value === guardrail.kind);
  const label = kind ? t(kind.label) : guardrail.kind;

  return (
    <p className="text-2xs text-muted-foreground">
      {label}{" "}
      {guardrail.applied
        ? t(
            "applied — {0} became {1}.",
            formatCurrency(Number(guardrail.raw ?? 0), currency),
            formatCurrency(Number(guardrail.result ?? 0), currency),
          )
        : t("did not apply.")}
    </p>
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
