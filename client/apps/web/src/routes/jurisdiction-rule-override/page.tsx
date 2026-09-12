import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/jurisdiction-rule-override-table"));

export function JurisdictionRuleOverridesPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Carrier Overrides")}
        description={t(
          "Hold your fleet to stricter limits than a state requires. An override can only tighten a limit, never loosen one, and applies to your organization alone.",
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
