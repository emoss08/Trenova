import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { AccountingAuthorizationCallback } from "../_components/accounting/accounting-authorization-callback";
import { xeroVendor } from "../_components/accounting/accounting-vendors";

export function XeroCallbackPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Connecting Xero"),
        description: t("Trenova finishes the connection Xero just approved."),
      }}
    >
      <AccountingAuthorizationCallback vendor={xeroVendor} />
    </PageLayout>
  );
}
