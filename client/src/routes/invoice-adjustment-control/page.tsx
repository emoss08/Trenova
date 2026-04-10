import { QueryLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { queries } from "@/lib/queries";
import { lazy } from "react";

const InvoiceAdjustmentControlForm = lazy(
  () => import("./_components/invoice-adjustment-control-form"),
);

export function InvoiceAdjustmentControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Invoice Adjustment Controls"
        description="Configure organization policy for credits, rebills, write-offs, and invoice adjustment review."
      />
      <div className="p-4">
        <QueryLazyComponent queryKey={queries.invoiceAdjustmentControl.get._def}>
          <InvoiceAdjustmentControlForm />
        </QueryLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
