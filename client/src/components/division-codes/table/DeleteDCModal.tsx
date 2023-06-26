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

import axios from "@/lib/AxiosConfig";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Box, Button, Modal, Text } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import React from "react";
import { useQueryClient } from "react-query";
import { divisionCodeTableStore } from "@/stores/AccountingStores";
import { DivisionCode } from "@/types/apps/accounting";

export const DeleteDCModal: React.FC = () => {
  const [loading, setLoading] = React.useState<boolean>(false);
  const [showDeleteRecordModal, setShowDeleteRecordModal] =
    divisionCodeTableStore.use("deleteModalOpen");
  const [selectedRecord] = divisionCodeTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const handleDelete = async (divisionCode: DivisionCode) => {
    setLoading(true);
    try {
      const response = await axios.delete(`division_codes/${divisionCode.id}`);
      if (response.status === 204) {
        queryClient
          .invalidateQueries(["division-codes-table-data"])
          .then(() => {
            notifications.show({
              title: "Record deleted",
              message: "Record has been deleted successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
          });
      }
    } catch (error: any) {
      console.log(error);
    } finally {
      setLoading(false);
      setShowDeleteRecordModal(false);
    }
  };

  if (!selectedRecord) return null;

  return (
    <>
      <Modal.Root
        opened={showDeleteRecordModal}
        onClose={() => setShowDeleteRecordModal(false)}
        centered
      >
        <Modal.Overlay />
        <Modal.Content>
          <Modal.Header>
            Please confirm your action
            <Modal.CloseButton />
          </Modal.Header>
          <Modal.Body>
            <Text size="sm">
              This action is irreversible and will permanently remove all data
              associated with this record. If you proceed, there will be no way
              to recover this user's information. Are you sure you want to
              proceed?
            </Text>
            <Box
              mt={10}
              style={{
                display: "flex",
                justifyContent: "flex-end",
              }}
            >
              <Button
                onClick={() => setShowDeleteRecordModal(false)}
                variant="default"
                mr={10}
              >
                No don't delete it
              </Button>
              <Button
                type="submit"
                color="red"
                variant="filled"
                ml={5}
                loading={loading}
                onClick={() => handleDelete(selectedRecord)}
              >
                Delete Record
              </Button>
            </Box>
          </Modal.Body>
        </Modal.Content>
      </Modal.Root>
    </>
  );
};
