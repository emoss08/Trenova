import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Customer } from "@trenova/shared/types/customer";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The customer's open receivables as the credit review last computed them.
 * They are kept by the system from the receivables ledger, so they are shown
 * rather than edited, beside the share of the limit they use.
 */
export function CreditExposureSummary() {
  const t = useT();
  const { control } = useFormContext<Customer>();
  const [balance, limit, reviewedAt, automatic, status, currency, reason] = useWatch({
    control,
    name: [
      "billingProfile.creditBalance",
      "billingProfile.creditLimit",
      "billingProfile.creditReviewedAt",
      "billingProfile.creditHoldAutomatic",
      "billingProfile.creditStatus",
      "billingProfile.billingCurrency",
      "billingProfile.creditHoldReason",
    ],
  });

  const open = Number(balance ?? 0);
  const ceiling = limit != null ? Number(limit) : null;
  const share = ceiling && ceiling > 0 ? Math.round((open / ceiling) * 100) : null;
  const placedAutomatically = automatic && (status === "Hold" || status === "Warning");

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium">{t("Open receivables")}</span>
      <div className="flex items-baseline gap-2">
        <span className="text-lg tabular-nums">{formatCurrency(open, currency || "USD")}</span>
        {share !== null ? (
          <span className="text-muted-foreground text-xs tabular-nums">
            {t("{0}% of the limit", share)}
          </span>
        ) : null}
      </div>
      <span className="text-muted-foreground text-xs">
        {reviewedAt
          ? t(
              "Open invoices less unapplied cash and credits, as of {0}. Kept up to date as invoices post and payments apply.",
              formatUnixDateTimeMedium(reviewedAt),
            )
          : t(
              "Computed from open invoices once this customer's receivables next move, or overnight.",
            )}
      </span>
      {placedAutomatically ? (
        <Alert size="sm" variant={status === "Hold" ? "warning" : "info"}>
          <AlertDescription>
            {status === "Hold"
              ? t(
                  "Held automatically: {0}. The hold lifts once the balance is paid back under the limit.",
                  reason,
                )
              : t("Warned automatically: {0}.", reason)}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}
