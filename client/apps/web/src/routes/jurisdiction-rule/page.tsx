import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/jurisdiction-rule-table"));

export function JurisdictionRulesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Jurisdiction Rules"),
        description: t(
          "Oversize and overweight limits per state. These limits are shared across every organization; record a carrier override to hold your own fleet to something stricter.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
