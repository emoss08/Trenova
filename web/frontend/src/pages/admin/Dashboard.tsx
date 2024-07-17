/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
