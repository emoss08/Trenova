import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { CalculationTrace, TraceStep } from "@trenova/shared/types/detention";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";

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
    return new Date(step.timestamp * 1000).toLocaleString();
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
 * with the contract term that produced it. Copyable, because ending a detention
 * dispute usually means pasting the derivation into a reply.
 */
export function CalculationReceipt({
  trace,
  currency = "USD",
  className,
}: CalculationReceiptProps) {
  const { copy, isCopied } = useCopyToClipboard();

  if (!trace || trace.steps.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        No calculation has been recorded for this stop yet.
      </p>
    );
  }

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h4 className="text-sm font-medium">How this charge was calculated</h4>
          <Badge variant="outline" className="text-[10px]">
            engine {trace.engineVersion}
          </Badge>
        </div>
        <Button
          size="sm"
          variant="outline"
          className="h-7 text-xs"
          onClick={() => void copy(receiptText(trace, currency))}
        >
          {isCopied ? "Copied" : "Copy"}
        </Button>
      </div>

      <ol className="flex flex-col">
        {trace.steps.map((step, index) => {
          const value = stepValue(step, currency);
          const isFinal = step.kind === "Final";

          return (
            <li
              key={`${step.kind}-${index}`}
              className={cn(
                "flex items-start gap-3 border-l-2 py-2 pl-3",
                isFinal
                  ? "border-l-foreground"
                  : step.isReduction
                    ? "border-l-red-400/60"
                    : "border-l-border",
              )}
            >
              <div className="min-w-0 flex-1">
                <p className={cn("text-sm", isFinal && "font-semibold")}>
                  {step.label}
                </p>
                {step.detail && (
                  <p className="text-muted-foreground text-xs">{step.detail}</p>
                )}
                {step.term && (
                  <p className="text-muted-foreground/80 mt-0.5 text-[11px] italic">
                    {step.term}
                  </p>
                )}
              </div>

              {value && (
                <span
                  className={cn(
                    "shrink-0 text-sm tabular-nums",
                    isFinal && "font-semibold",
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
