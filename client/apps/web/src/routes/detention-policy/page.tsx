import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/detention-policy-table"));

export function DetentionPolicyPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Detention Policies"),
        description: t(
          "Encode each contract's detention terms and see what they would charge before they touch a shipment",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
