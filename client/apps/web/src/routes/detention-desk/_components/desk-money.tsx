import NumberFlow from "@number-flow/react";
import { cn } from "@trenova/shared/lib/utils";

type DeskMoneyProps = {
  value: number;
  currency?: string;
  /** Whole dollars keep a dense list scannable; cents matter on the headline figures. */
  precise?: boolean;
  className?: string;
};

/**
 * Money that moves. The desk repolls every thirty seconds and detention accrues
 * the whole time, so a figure that jumps silently reads as a render artifact —
 * animating the digits makes the number visibly *grow*, which is the point.
 */
export function DeskMoney({ value, currency, precise = true, className }: DeskMoneyProps) {
  return (
    <NumberFlow
      value={value}
      format={{
        style: "currency",
        currency: currency || "USD",
        minimumFractionDigits: precise ? 2 : 0,
        maximumFractionDigits: precise ? 2 : 0,
      }}
      className={cn("tabular-nums", className)}
    />
  );
}
