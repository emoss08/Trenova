import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { AccountingAuthorizationCallback } from "../_components/accounting/accounting-authorization-callback";
import { quickBooksVendor } from "../_components/accounting/accounting-vendors";

export function QuickBooksCallbackPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Connecting QuickBooks Online"),
        description: t("Trenova finishes the connection QuickBooks Online just approved."),
      }}
    >
      <AccountingAuthorizationCallback vendor={quickBooksVendor} />
    </PageLayout>
  );
}
