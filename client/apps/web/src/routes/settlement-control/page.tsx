import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const SettlementControlForm = lazy(() => import("./_components/settlement-control-form"));

export function SettlementControlPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Settlement Control")}
        description={t("Configure pay periods, accrual triggers, exception policies, and escrow interest")}
      />
      <SuspenseLoader>
        <div className="p-4">
          <SettlementControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
