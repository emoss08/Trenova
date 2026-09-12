import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DataEntryControlForm = lazy(() => import("./_components/data-entry-control-form"));

export function DataEntryControlPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Data Entry Control")}
        description={t("Configure case formatting rules for data entry across the system")}
      />
      <div className="p-4">
        <SuspenseLoader>
          <DataEntryControlForm />
        </SuspenseLoader>
      </div>
    </AdminPageLayout>
  );
}
