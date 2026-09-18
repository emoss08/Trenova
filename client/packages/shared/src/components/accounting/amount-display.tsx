import { cn, formatCurrency } from "@trenova/shared/lib/utils";

type AmountDisplayProps = {
  value: number;
  currency?: string;
  variant?: "neutral" | "positive" | "negative" | "auto";
  className?: string;
};

export function AmountDisplay({
  value,
  currency = "USD",
  variant = "neutral",
  className,
}: AmountDisplayProps) {
  const displayValue = value / 100;
  const resolvedVariant = variant === "auto" ? (value >= 0 ? "positive" : "negative") : variant;

  return (
    <span
      className={cn(
        "tabular-nums",
        resolvedVariant === "positive" && "text-success-foreground",
        resolvedVariant === "negative" && "text-danger-foreground",
        className,
      )}
    >
      {formatCurrency(displayValue, currency)}
    </span>
  );
}
