import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { AccountingAuthorizationCallback } from "../_components/accounting/accounting-authorization-callback";
import { businessCentralVendor } from "../_components/accounting/accounting-vendors";

export function BusinessCentralCallbackPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Connecting Business Central"),
        description: t("Trenova finishes the connection Microsoft just approved."),
      }}
    >
      <AccountingAuthorizationCallback vendor={businessCentralVendor} />
    </PageLayout>
  );
}
