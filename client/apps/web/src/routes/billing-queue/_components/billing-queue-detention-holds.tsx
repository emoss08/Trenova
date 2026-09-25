import { recordPath } from "@/config/record-links";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { DetentionHold } from "@trenova/shared/types/billing-queue";
import type { BillingHoldReason } from "@trenova/shared/types/detention";
import { TimerIcon } from "lucide-react";
import { Link } from "react-router";

function holdReasonLabel(t: TranslateFn, reason: BillingHoldReason): string {
  switch (reason) {
    case "OverApprovalThreshold":
      return t("Over the approval threshold");
    case "NoticeNotSent":
      return t("Notice not sent in time");
    case "Escalated":
      return t("Escalated for review");
  }
}

/**
 * The detention charges keeping this item from being approved. Each links to
 * the charge on the detention desk, where it is approved or waived; the approve
 * action stays off until none is left.
 */
export function BillingQueueDetentionHolds({ holds }: { holds: DetentionHold[] }) {
  const t = useT();

  if (holds.length === 0) {
    return null;
  }

  return (
    <div className="shrink-0 px-4 pt-2">
      <Alert size="sm" variant="warning" data-testid="billing-queue-detention-holds">
        <TimerIcon />
        <AlertTitle>
          {t(
            "{0, plural, one {# detention charge needs approval} other {# detention charges need approval}}",
            holds.length,
          )}
        </AlertTitle>
        <AlertDescription>
          <p>
            {t(
              "This item cannot be approved until each charge is approved or waived on the detention desk.",
            )}
          </p>
          <ul className="grid w-full gap-0.5">
            {holds.map((hold) => (
              <li
                key={hold.occurrenceId}
                className="flex flex-wrap items-baseline justify-between gap-x-3"
              >
                <Link
                  to={recordPath("detention_occurrence", hold.occurrenceId)}
                  className="ui-focus-ring text-brand rounded-control underline-offset-2 hover:underline"
                >
                  {hold.locationName || t("Detention charge")}
                </Link>
                <span className="flex items-baseline gap-2">
                  <span className="tabular-nums">
                    {formatCurrency(hold.billableAmount ?? 0, hold.currency)}
                  </span>
                  <span>{holdReasonLabel(t, hold.reason)}</span>
                </span>
              </li>
            ))}
          </ul>
        </AlertDescription>
      </Alert>
    </div>
  );
}
