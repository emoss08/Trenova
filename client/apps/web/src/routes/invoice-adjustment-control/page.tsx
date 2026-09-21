import { useT } from "@trenova/shared/i18n/use-t";
import { QueryLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { lazy } from "react";

const InvoiceAdjustmentControlForm = lazy(
  () => import("./_components/invoice-adjustment-control-form"),
);

export function InvoiceAdjustmentControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Invoice Adjustment Controls"),
        description: t(
          "Configure organization policy for credits, rebills, write-offs, and invoice adjustment review.",
        ),
      }}
    >
      <QueryLazyComponent queryKey={queries.invoiceAdjustmentControl.get._def}>
        <InvoiceAdjustmentControlForm />
      </QueryLazyComponent>
    </PageLayout>
  );
}
