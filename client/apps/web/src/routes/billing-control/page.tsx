import { useT } from "@trenova/shared/i18n/use-t";
import { QueryLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { TriangleAlertIcon } from "lucide-react";
import { lazy } from "react";

const BillingControlForm = lazy(() => import("./_components/billing-control-form"));

export function BillingControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Billing Control"),
        description: t("Configure and manage your billing control settings"),
      }}
    >
      <BillingControlAlert />
      <QueryLazyComponent queryKey={queries.billingControl.get._def}>
        <BillingControlForm />
      </QueryLazyComponent>
    </PageLayout>
  );
}

function BillingControlAlert() {
  const t = useT();

  return (
    <Alert variant="warning" size="sm">
      <TriangleAlertIcon />
      <AlertTitle>{t("Critical Financial Configuration")}</AlertTitle>
      <AlertDescription>
        {t(
          "Billing Control settings directly impact your organization's revenue processing, financial reporting, and customer invoicing. Changes to these settings should be made infrequently and only after thorough review by financial stakeholders.",
        )}
      </AlertDescription>
    </Alert>
  );
}
