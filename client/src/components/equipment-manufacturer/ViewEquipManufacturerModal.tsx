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
import { Box, Button, Group, Modal } from "@mantine/core";
import { EquipmentManufacturer } from "@/types/equipment";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { useEquipManufacturerTableStore as store } from "@/stores/EquipmentStore";
import { ViewTextInput } from "../common/fields/TextInput";
import { ViewTextarea } from "@/components/common/fields/TextArea";

function ModalBody({
  equipManufacturer,
}: {
  equipManufacturer: EquipmentManufacturer;
}) {
  const { classes } = useFormStyles();

  return (
    <>
      <Box className={classes.div}>
        <ViewTextInput
          value={equipManufacturer.name}
          label="Name"
          placeholder="Name"
          description="Unique name for the equipment manufacturer."
        />
        <ViewTextarea
          value={equipManufacturer.description || ""}
          label="Description"
          description="Description of the equipment manufacturer."
          placeholder="Description"
        />
      </Box>
      <Group position="right" mt="md">
        <Button
          color="white"
          type="submit"
          className={classes.control}
          onClick={() => {
            store.set("viewModalOpen", false);
            store.set("editModalOpen", true);
          }}
        >
          Edit Equipment Manufacturer
        </Button>
      </Group>
    </>
  );
}

export function ViewEMModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [equipManufacturer] = store.use("selectedRecord");

  return (
    <Modal.Root
      opened={showViewModal}
      onClose={() => setShowViewModal(false)}
      size="md"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Equipment Manufacturer</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {equipManufacturer && (
            <ModalBody equipManufacturer={equipManufacturer} />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
