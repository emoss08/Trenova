import { QueryLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { queries } from "@/lib/queries";
import { TriangleAlertIcon } from "lucide-react";
import { lazy } from "react";

const BillingControlForm = lazy(
  () => import("./_components/billing-control-form"),
);

export function BillingControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Billing Control"
        description="Configure and manage your billing control settings"
      />
      <BillingControlAlert />
      <QueryLazyComponent queryKey={queries.billingControl.get._def}>
        <BillingControlForm />
      </QueryLazyComponent>
    </AdminPageLayout>
  );
}

function BillingControlAlert() {
  return (
    <div className="mb-4 flex w-full items-center justify-between rounded-md border border-amber-600/50 bg-amber-500/10 p-4">
      <div className="flex w-full items-center gap-3 text-amber-600">
        <TriangleAlertIcon className="size-5" />
        <div className="flex flex-col">
          <p className="text-sm font-semibold">
            Critical Financial Configuration
          </p>
          <p className="text-xs">
            Billing Control settings directly impact your organization&apos;s
            revenue processing, financial reporting, and customer invoicing.
            Changes to these settings should be made infrequently and only after
            thorough review by financial stakeholders.
          </p>
        </div>
      </div>
    </div>
  );
}
