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
import { chargeTypeTableStore } from "@/stores/BillingStores";
import { getChargeTypeDetails } from "@/requests/BillingRequestFactory";
import { EditChargeTypeModalForm } from "@/components/charge-types/table/_partials/EditChargeTypeModalForm";

export const EditChargeTypeModal: React.FC = () => {
  const [showEditModal, setShowEditModal] =
    chargeTypeTableStore.use("editModalOpen");
  const [chargeType] = chargeTypeTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: chargeTypeData, isLoading: isChargeTypeDataLoading } = useQuery(
    {
      queryKey: ["chargeType", chargeType?.id],
      queryFn: () => {
        if (!chargeType) {
          return Promise.resolve(null);
        }
        return getChargeTypeDetails(chargeType.id);
      },
      enabled: showEditModal,
      initialData: () => {
        return queryClient.getQueryData(["chargeType", chargeType?.id]);
      },
    }
  );

  if (!showEditModal) return null;

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Charge Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {isChargeTypeDataLoading ? (
            <Stack>
              <Skeleton height={400} />
            </Stack>
          ) : (
            <>
              {chargeTypeData && (
                <EditChargeTypeModalForm chargeType={chargeTypeData} />
              )}
            </>
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
};
