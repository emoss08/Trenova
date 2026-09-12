import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/detention-policy-table"));

export function DetentionPolicyPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Detention Policies")}
        description={t(
          "Encode each contract's detention terms and see what they would charge before they touch a shipment",
        )}
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
