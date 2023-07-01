/*
 * COPYRIGHT(c) 2023 MONTA
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

import React from "react";
import { generalLedgerTableStore } from "@/stores/AccountingStores";
import { MontaTable } from "@/components/MontaTable";
import { GLAccountTableColumns } from "@/components/gl-accounts/table/GLAccountTableColumns";
import { GLAccountTableTopToolbar } from "@/components/gl-accounts/table/GLAccountTableTopToolbar";
import { EditGLAccountModal } from "@/components/gl-accounts/table/EditGLAccountModal";
import { ExportGLAccountModal } from "@/components/gl-accounts/table/ExportGLAccountModal";
import { ViewGLAccountModal } from "@/components/gl-accounts/table/ViewGLAccountModal";
import { CreateGLAccountModal } from "./CreateGLAccountModal";

export const GLAccountTable = () => {
  return (
    <MontaTable
      store={generalLedgerTableStore}
      link="/gl_accounts"
      columns={GLAccountTableColumns}
      TableTopToolbar={GLAccountTableTopToolbar}
      TableEditModal={EditGLAccountModal}
      TableExportModal={ExportGLAccountModal}
      TableViewModal={ViewGLAccountModal}
      displayDeleteModal={true}
      TableCreateDrawer={CreateGLAccountModal}
      queryKey="gl-account-table-data"
      queryKey2="glAccounts"
    />
  );
};
