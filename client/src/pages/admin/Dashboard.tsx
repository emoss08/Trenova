import AdminLayout from "@/components/admin-page/layout";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { useUserOrganization } from "@/hooks/useQueries";
import { type Organization } from "@/types/organization";
import { lazy } from "react";

const GeneralPage = lazy(() => import("@/components/admin-page/general-page"));

export default function AdminPage() {
  const { userOrganizationData, userOrganizationError } = useUserOrganization();
  if (userOrganizationError) {
    return (
      <ErrorLoadingData message="An Error occurred, while loading your profile, plese contact your system administrator." />
    );
  }

  return (
    <AdminLayout>
      {userOrganizationData && (
        <GeneralPage organization={userOrganizationData as Organization} />
      )}
    </AdminLayout>
  );
}
