import AdminLayout from "@/components/admin-page/layout";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { ComponentLoader } from "@/components/ui/component-loader";
import { useOrganization } from "@/hooks/useQueries";
import { type Organization } from "@/types/organization";
import { lazy } from "react";

const GeneralPage = lazy(() => import("@/components/admin-page/general-page"));

export default function AdminPage() {
  const { organizationData, organizationError, organizationLoading } =
    useOrganization();
  if (organizationError) {
    return (
      <ErrorLoadingData message="An Error occurred, while loading your profile, plese contact your system administrator." />
    );
  }

  return (
    <AdminLayout>
      {organizationLoading ? (
        <ComponentLoader className="h-[40vh]" />
      ) : (
        <GeneralPage organization={organizationData as Organization} />
      )}
    </AdminLayout>
  );
}
