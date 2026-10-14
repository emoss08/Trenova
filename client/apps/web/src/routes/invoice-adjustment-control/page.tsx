import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { InvoiceAdjustmentControlSkeleton } from "./skeleton";

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
      <LazyComponent fallback={<InvoiceAdjustmentControlSkeleton />}>
        <InvoiceAdjustmentControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
