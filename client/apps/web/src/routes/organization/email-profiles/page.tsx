import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { PurposeAssignmentsPanel } from "./_components/purpose-assignments-panel";

const Table = lazy(() => import("./_components/email-profile-table"));

export function EmailProfilesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Email profiles"),
        description: t(
          "Manage verified sender identities and route email purposes to the right provider profile.",
        ),
      }}
    >
      <PurposeAssignmentsPanel />
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
