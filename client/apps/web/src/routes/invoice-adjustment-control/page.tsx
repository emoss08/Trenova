import { useT } from "@trenova/shared/i18n/use-t";
import { QueryLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { invoiceAdjustmentControl } from "@/lib/queries/invoice-adjustment-control";
import { lazy } from "react";

const InvoiceAdjustmentControlForm = lazy(
  () => import("./_components/invoice-adjustment-control-form"),
);

export function InvoiceAdjustmentControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Invoice adjustment controls"),
        description: t(
          "Configure organization policy for credits, rebills, write-offs, and invoice adjustment review.",
        ),
      }}
    >
      <QueryLazyComponent queryKey={invoiceAdjustmentControl.get._def}>
        <InvoiceAdjustmentControlForm />
      </QueryLazyComponent>
    </PageLayout>
  );
}
