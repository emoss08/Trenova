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
import { useForm, yupResolver } from "@mantine/form";
import { Box, Button, Group, Modal } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import {
  EquipmentManufacturer,
  EquipmentManufacturerFormValues as FormValues,
} from "@/types/equipment";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { equipManufacturerSchema } from "@/lib/schemas/EquipmentSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useEquipManufacturerTableStore as store } from "@/stores/EquipmentStore";
import { TableStoreProps } from "@/types/tables";

function ModalBody({
  equipManufacturer,
}: {
  equipManufacturer: EquipmentManufacturer;
}) {
  const [loading, setLoading] = React.useState<boolean>(false);
  const { classes } = useFormStyles();

  const form = useForm<FormValues>({
    validate: yupResolver(equipManufacturerSchema),
    initialValues: {
      name: equipManufacturer.name,
      description: equipManufacturer.description,
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    Omit<TableStoreProps<EquipmentManufacturer>, "drawerOpen">
  >(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/equipment_manufacturers/${equipManufacturer.id}/`,
      successMessage: "Equipment Manufacturer updated successfully.",
      queryKeysToInvalidate: ["equipment-manufacturer-table-data"],
      additionalInvalidateQueries: ["equipmentManufacturers"],
      closeModal: true,
      errorMessage: "Failed to update equipment manufacturer.",
    },
    () => setLoading(false),
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <ValidatedTextInput<FormValues>
          form={form}
          name="name"
          label="Name"
          placeholder="Name"
          description="Unique name for the equipment manufacturer."
          withAsterisk
        />
        <ValidatedTextArea<FormValues>
          form={form}
          name="description"
          label="Description"
          description="Description of the equipment manufacturer."
          placeholder="Description"
        />
      </Box>
      <Group position="right" mt="md">
        <Button type="submit" className={classes.control} loading={loading}>
          Submit
        </Button>
      </Group>
    </form>
  );
}

export function EditEMModal(): React.ReactElement {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [equipManufacturer] = store.use("selectedRecord");

  return (
    <Modal.Root
      opened={showEditModal}
      onClose={() => setShowEditModal(false)}
      size="md"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Equipment Manufacturer</Modal.Title>
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
