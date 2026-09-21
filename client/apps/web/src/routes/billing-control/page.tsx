import { useT } from "@trenova/shared/i18n/use-t";
import { QueryLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
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
    <div className="mb-4 flex w-full items-center justify-between rounded-md border border-warning/50 bg-warning/10 p-4">
      <div className="flex w-full items-center gap-3 text-warning-foreground">
        <TriangleAlertIcon className="size-5" />
        <div className="flex flex-col">
          <p className="text-sm font-semibold">{t("Critical Financial Configuration")}</p>
          <p className="text-xs">
            {t(
              "Billing Control settings directly impact your organization's revenue processing, financial reporting, and customer invoicing. Changes to these settings should be made infrequently and only after thorough review by financial stakeholders.",
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
