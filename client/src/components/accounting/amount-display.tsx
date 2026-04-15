import { cn, formatCurrency } from "@/lib/utils";

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
        resolvedVariant === "positive" && "text-green-600 dark:text-green-400",
        resolvedVariant === "negative" && "text-red-600 dark:text-red-400",
        className,
      )}
    >
      {formatCurrency(displayValue, currency)}
    </span>
  );
}
