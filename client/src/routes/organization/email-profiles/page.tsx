import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";
import { PurposeAssignmentsPanel } from "./_components/purpose-assignments-panel";

const Table = lazy(() => import("./_components/email-profile-table"));

export function EmailProfilesPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Email Profiles"
        description="Manage verified sender identities and route email purposes to the right provider profile."
      />
      <div className="flex flex-col gap-4 p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
        <PurposeAssignmentsPanel />
      </div>
    </AdminPageLayout>
  );
}
