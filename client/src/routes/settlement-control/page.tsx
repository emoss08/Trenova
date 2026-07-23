import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const SettlementControlForm = lazy(() => import("./_components/settlement-control-form"));

export function SettlementControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Settlement Control"
        description="Configure pay periods, accrual triggers, exception policies, and escrow interest"
      />
      <SuspenseLoader>
        <div className="p-4">
          <SettlementControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
