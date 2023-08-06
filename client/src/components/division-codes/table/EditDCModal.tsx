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

import { Modal, Skeleton } from "@mantine/core";
import React, { Suspense } from "react";
import { useQuery, useQueryClient } from "react-query";
import { divisionCodeTableStore } from "@/stores/AccountingStores";
import { EditDCModalForm } from "@/components/division-codes/table/_Partials/EditDCModalForm";
import { getGLAccounts } from "@/requests/AccountingRequestFactory";
import { GeneralLedgerAccount } from "@/types/apps/accounting";

export const EditDCModal: React.FC = () => {
  const [showEditModal, setShowEditModal] =
    divisionCodeTableStore.use("editModalOpen");
  const [divisionCode] = divisionCodeTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: glAccountData } = useQuery({
    queryKey: "gl-account-data",
    queryFn: () => getGLAccounts(),
    enabled: showEditModal,
    initialData: () => queryClient.getQueryData("gl-account"),
    staleTime: Infinity,
  });

  const selectGlAccountData =
    glAccountData?.map((glAccount: GeneralLedgerAccount) => ({
      value: glAccount.id,
      label: glAccount.account_number,
    })) || [];

  if (!showEditModal) return null;

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Division Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            {divisionCode && (
              <EditDCModalForm
                divisionCode={divisionCode}
                selectGlAccountData={selectGlAccountData}
              />
            )}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
};
