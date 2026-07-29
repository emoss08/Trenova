import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { Button } from "@trenova/shared/components/ui/button";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { CalculationTrace, TraceStep } from "@trenova/shared/types/detention";
import { formatUnixDateTime } from "@trenova/shared/lib/date";

type CalculationReceiptProps = {
  trace: CalculationTrace | null | undefined;
  currency?: string;
  className?: string;
};

function stepValue(step: TraceStep, currency: string): string | null {
  if (step.amount !== null && step.amount !== undefined) {
    return formatCurrency(step.amount, currency);
  }

  if (step.minutes !== null && step.minutes !== undefined) {
    return formatDetentionMinutes(step.minutes);
  }

  if (step.timestamp !== null && step.timestamp !== undefined) {
    return formatUnixDateTime(step.timestamp);
  }

  return null;
}

function receiptText(trace: CalculationTrace, currency: string): string {
  return trace.steps
    .map((step) => {
      const value = stepValue(step, currency);
      const detail = step.detail ? `: ${step.detail}` : "";
      const term = step.term ? ` [${step.term}]` : "";
      const amount = value ? ` — ${value}` : "";
      return `• ${step.label}${detail}${term}${amount}`;
    })
    .join("\n");
}

/**
 * The calculation receipt: every step from arrival to final charge, annotated
 * with the contract term that produced it. Set as a ledger — label left, value
 * right — because ending a detention dispute usually means pasting the
 * derivation into a reply and having it read like an invoice.
 */
export function CalculationReceipt({
  trace,
  currency = "USD",
  className,
}: CalculationReceiptProps) {
  const { copy, isCopied } = useCopyToClipboard();

  if (!trace || trace.steps.length === 0) {
    return (
      <p className="text-xs text-muted-foreground">
        No calculation has been recorded for this stop yet.
      </p>
    );
  }

  return (
    <div className={cn("flex flex-col", className)}>
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          How this charge was calculated
        </h4>
        <div className="flex items-center gap-2">
          <span className="text-2xs text-muted-foreground/70 tabular-nums">
            engine {trace.engineVersion}
          </span>
          <Button
            size="xxs"
            variant="ghost"
            className="text-2xs text-muted-foreground hover:text-foreground"
            onClick={() => void copy(receiptText(trace, currency))}
          >
            {isCopied ? "Copied" : "Copy"}
          </Button>
        </div>
      </div>

      <ol className="mt-2 divide-y divide-border/60">
        {trace.steps.map((step, index) => {
          const value = stepValue(step, currency);
          const isFinal = step.kind === "Final";

          return (
            <li
              key={`${step.kind}-${index}`}
              className={cn(
                "flex items-baseline justify-between gap-3 py-2",
                isFinal && "border-t border-border pt-2.5",
              )}
            >
              <div className="min-w-0">
                <p className={cn("text-xs", isFinal && "font-medium")}>{step.label}</p>
                {step.detail && <p className="text-2xs text-muted-foreground">{step.detail}</p>}
                {step.term && <p className="text-2xs text-muted-foreground/70">{step.term}</p>}
              </div>

              {value && (
                <span
                  className={cn(
                    "shrink-0 text-xs tabular-nums",
                    isFinal ? "font-medium" : "text-muted-foreground",
                    step.isReduction && "text-red-600 dark:text-red-400",
                  )}
                >
                  {step.isReduction ? "−" : ""}
                  {value}
                </span>
              )}
            </li>
          );
        })}
      </ol>
    </div>
  );
}
