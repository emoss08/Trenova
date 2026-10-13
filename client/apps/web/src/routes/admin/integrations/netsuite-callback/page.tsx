import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { AccountingAuthorizationCallback } from "../_components/accounting/accounting-authorization-callback";
import { netSuiteVendor } from "../_components/accounting/accounting-vendors";

export function NetSuiteCallbackPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Connecting NetSuite"),
        description: t("Trenova finishes the connection NetSuite just approved."),
      }}
    >
      <AccountingAuthorizationCallback vendor={netSuiteVendor} />
    </PageLayout>
  );
}
