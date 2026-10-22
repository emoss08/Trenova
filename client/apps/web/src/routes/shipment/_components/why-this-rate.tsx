import { useT } from "@trenova/shared/i18n/use-t";
import { BillView } from "@/components/tariff-bill/bill-view";
import { rateQuoteOutcomeChoices } from "@/lib/choices";
import { rateQuote } from "@/lib/queries/rate-quote";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import type { RateQuote, RateTraceCandidate } from "@trenova/shared/types/rate";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import { InfoCircleIcon } from "@trenova/shared/components/icons";

type WhyThisRateProps = {
  shipmentId?: string;
};

/**
 * The answer to "why did I get this rate", which is the complaint every
 * carrier and broker has about the system they are leaving.
 *
 * It shows the contract and lane that won, its tariff's bill line by line (charge labels as
 * the tariff wrote them, adjustments in the reader's words), and —
 * the part that actually settles arguments — every rate that was considered and
 * the reason it lost.
 */
export function WhyThisRate({ shipmentId }: WhyThisRateProps) {
  const t = useT();

  const { data: quote, isLoading } = useQuery({
    ...rateQuote.appliedForShipment(shipmentId ?? ""),
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
            <InfoCircleIcon className="size-3" />
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
          <Badge variant="neutral" appearance="outline" className="text-2xs">
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

      {trace?.refusal ? (
        <div className="border-b p-3">
          <p className="text-muted-foreground mb-1 text-xs font-medium">
            {t("The tariff refused to rate")}
          </p>
          <p className="text-2xs">{trace.refusal}</p>
        </div>
      ) : null}

      {trace?.bill && (
        <div className="border-b p-3">
          <p className="text-muted-foreground mb-2 text-xs font-medium">
            {trace.templateName
              ? t("What made up the rate · {0}", trace.templateName)
              : t("What made up the rate")}
          </p>
          <BillView bill={trace.bill} currency={trace.bill.currency || quote.currency} />
        </div>
      )}

      {Object.keys(trace?.tariffInputs ?? {}).length > 0 && (
        <div className="border-b p-3">
          <p className="text-muted-foreground mb-1 text-xs font-medium">{t("Terms it read")}</p>
          <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs">
            {Object.entries(trace?.tariffInputs ?? {}).map(([name, value]) => (
              <div key={name} className="contents">
                <dt className="text-muted-foreground">{humanizeTariffName(name)}</dt>
                <dd className="font-mono tabular-nums">
                  {formatTariffValue(value as TariffWireValue, t, {
                    currency: trace?.bill?.currency || quote.currency,
                  })}
                </dd>
              </div>
            ))}
          </dl>
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
          <p className="text-xs text-muted-foreground mb-2 font-medium">
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

function LoserRow({ candidate }: { candidate: RateTraceCandidate }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs">{candidate.ruleLabel || candidate.laneKey}</span>
      <span className="text-2xs text-muted-foreground">{candidate.rejectDetail}</span>
    </div>
  );
}
