import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";
import { PurposeAssignmentsPanel } from "./_components/purpose-assignments-panel";

const Table = lazy(() => import("./_components/email-profile-table"));

export function EmailProfilesPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Email Profiles")}
        description={t(
          "Manage verified sender identities and route email purposes to the right provider profile.",
        )}
      />
      <div className="flex flex-col gap-4 p-4">
        <PurposeAssignmentsPanel />
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
