import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

type ExchangeRateLineProps = {
  currency: string;
  rate: string | null | undefined;
  quotedOn: number | null | undefined;
  paidRate?: string | null;
  paidQuotedOn?: number | null;
  className?: string;
};

const RATE_FORMAT = new Intl.NumberFormat(undefined, { maximumFractionDigits: 6 });

export function formatExchangeRate(rate: string): string {
  const parsed = Number(rate);
  return Number.isFinite(parsed) ? RATE_FORMAT.format(parsed) : rate;
}

export function ExchangeRateLine({
  currency,
  rate,
  quotedOn,
  paidRate,
  paidQuotedOn,
  className,
}: ExchangeRateLineProps) {
  const t = useT();
  if (!rate && !paidRate) {
    return null;
  }
  const parts: string[] = [];
  if (rate) {
    parts.push(
      t(
        "1 {0} = {1}, quoted {2}",
        currency,
        formatExchangeRate(rate),
        formatUnixDateMedium(quotedOn),
      ),
    );
  }
  if (paidRate) {
    parts.push(
      t(
        "paid at {0}, quoted {1}",
        formatExchangeRate(paidRate),
        formatUnixDateMedium(paidQuotedOn),
      ),
    );
  }
  return (
    <p className={cn("text-muted-foreground text-xs tabular-nums", className)}>
      {t("Exchange rate")} · {parts.join(" · ")}
    </p>
  );
}
