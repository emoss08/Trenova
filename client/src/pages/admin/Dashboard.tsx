/*
 * COPYRIGHT(c) 2024 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import AdminLayout from "@/components/admin-page/layout";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { getUserOrganizationId } from "@/lib/auth";
import { getUserOrganizationDetails } from "@/services/OrganizationRequestService";
import { QueryKeys } from "@/types";
import { Organization } from "@/types/organization";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { lazy } from "react";

const GeneralPage = lazy(() => import("@/components/admin-page/general-page"));

export default function AdminPage() {
  const queryClient = useQueryClient();

  // Get User organization data
  const organizationId = getUserOrganizationId() || "";

  const { data: organizationData, isError } = useQuery({
    queryKey: ["organization", organizationId] as QueryKeys[],
    queryFn: async () => {
      if (!organizationId) {
        return Promise.resolve(null);
      }
      return getUserOrganizationDetails();
    },
    initialData: (): Organization | undefined =>
      queryClient.getQueryData(["organization", organizationId]),
    staleTime: Infinity,
  });

  if (isError) {
    return (
      <ErrorLoadingData message="An Error occurred, while loading your profile, plese contact your system administrator." />
    );
  }

  return (
    <AdminLayout>
      {organizationData && (
        <GeneralPage organization={organizationData as Organization} />
      )}
    </AdminLayout>
  );
}
