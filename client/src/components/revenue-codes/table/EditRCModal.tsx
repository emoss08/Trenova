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

import { Modal, Skeleton, Stack } from "@mantine/core";
import React from "react";
import { useQuery, useQueryClient } from "react-query";
import { revenueCodeTableStore } from "@/stores/AccountingStores";
import {
  getGLAccounts,
  getRevenueCodeDetail,
} from "@/requests/AccountingRequestFactory";
import { GeneralLedgerAccount } from "@/types/apps/accounting";
import { EditRCModalForm } from "@/components/revenue-codes/table/_Partials/EditRCModalForm";

export const EditRCModal: React.FC = () => {
  const [showEditModal, setShowEditModal] =
    revenueCodeTableStore.use("editModalOpen");
  const [revenueCode] = revenueCodeTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: glAccountData, isLoading: isGLAccountDataLoading } = useQuery({
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

  const { data: revenueCodeData, isLoading: isRevenueCodeDataLoading } =
    useQuery({
      queryKey: ["revenueCode", revenueCode?.id],
      queryFn: () => {
        if (!revenueCode) {
          return Promise.resolve(null);
        }
        return getRevenueCodeDetail(revenueCode.id);
      },
      enabled: showEditModal,
      initialData: () => queryClient.getQueryData(["revenueCode", revenueCode?.id]),
      staleTime: Infinity, // Never refetch
    });

  if (!showEditModal) return null;

  const isDataLoading = isRevenueCodeDataLoading || isGLAccountDataLoading;

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Revenue Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {isDataLoading ? (
            <Stack>
              <Skeleton height={400} />
            </Stack>
          ) : (
            <>
              {revenueCodeData && (
                <EditRCModalForm
                  revenueCode={revenueCodeData}
                  selectGlAccountData={selectGlAccountData}
                />
              )}
            </>
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
};
