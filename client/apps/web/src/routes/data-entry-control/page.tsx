import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DataEntryControlForm = lazy(() => import("./_components/data-entry-control-form"));

export function DataEntryControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Data Entry Control"
        description="Configure case formatting rules for data entry across the system"
      />
      <div className="p-4">
        <SuspenseLoader>
          <DataEntryControlForm />
        </SuspenseLoader>
      </div>
    </AdminPageLayout>
  );
}
