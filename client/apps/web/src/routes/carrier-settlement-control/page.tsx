import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const CarrierSettlementControlForm = lazy(
  () => import("./_components/carrier-settlement-control-form"),
);

export function CarrierSettlementControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Carrier Settlement Control"
        description="Configure carrier pay periods, cost accrual triggers, batch automation, invoice-match tolerance, and AP posting accounts"
      />
      <SuspenseLoader>
        <div className="p-4">
          <CarrierSettlementControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
