import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/lading-partner-table"));

/** Every trading partner, each opening its Lading project in the studio. */
export function LadingProjectsPage() {
  const t = useT();
  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Lading studio"),
        description: t("Open a partner to write its maps, guides, rules and flows"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
