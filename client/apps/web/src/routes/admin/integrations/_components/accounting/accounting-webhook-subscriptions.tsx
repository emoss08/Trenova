import type { AccountingWebhookSubscriptionSummary } from "@/lib/graphql/accounting-sync";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { AccountingVendor } from "./accounting-vendors";

type AccountingWebhookSubscriptionsProps = {
  vendor: AccountingVendor;
  summary: AccountingWebhookSubscriptionSummary;
};

export function AccountingWebhookSubscriptions({
  vendor,
  summary,
}: AccountingWebhookSubscriptionsProps) {
  const t = useT();

  return (
    <div className="space-y-3 border-t pt-4">
      {!summary.configured ? (
        <Alert variant="info" size="sm">
          <AlertDescription>
            {t(
              "This server has no public https address for {0} to send change notices to, so Trenova reads changes every five minutes instead.",
              vendor.name,
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      {summary.configured && summary.failed > 0 ? (
        <Alert variant="warning" size="sm">
          <AlertTitle>
            {t(
              "{0} of Trenova's change subscriptions at {1} could not be kept",
              summary.failed,
              vendor.name,
            )}
          </AlertTitle>
          <AlertDescription>
            {summary.lastError
              ? t(
                  "{0} Trenova retries within the hour and reads changes every five minutes meanwhile.",
                  summary.lastError,
                )
              : t("Trenova retries within the hour and reads changes every five minutes meanwhile.")}
          </AlertDescription>
        </Alert>
      ) : null}
      <DescriptionList columns={2}>
        <DescriptionItem label={t("Change subscriptions")} numeric>
          {summary.active > 0
            ? t("{0} active", summary.active)
            : summary.pending > 0
              ? t("Being created")
              : <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Next renewal due")} numeric>
          {summary.nextExpiryAt ? (
            formatUnixDateMedium(summary.nextExpiryAt)
          ) : (
            <DescriptionEmpty />
          )}
        </DescriptionItem>
      </DescriptionList>
    </div>
  );
}
