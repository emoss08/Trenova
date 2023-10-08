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
import { Button, Drawer, Group } from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import {
  EquipmentManufacturer,
  EquipmentManufacturerFormValues as FormValues,
} from "@/types/equipment";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { useEquipManufacturerTableStore as store } from "@/stores/EquipmentStore";
import { equipManufacturerSchema } from "@/lib/validations/EquipmentSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { EquipmentManufacturerForm } from "./CreateEquipManfacturerModal";

function EditEquipManufacturerForm({
  equipManufacturer,
  onCancel,
}: {
  equipManufacturer: EquipmentManufacturer;
  onCancel: () => void;
}) {
  const [loading, setLoading] = React.useState<boolean>(false);
  const { classes } = useFormStyles();

  const form = useForm<FormValues>({
    validate: yupResolver(equipManufacturerSchema),
    initialValues: {
      status: equipManufacturer.status,
      name: equipManufacturer.name,
      description: equipManufacturer.description,
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    TableStoreProps<EquipmentManufacturer>
  >(
    form,
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
    store,
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <EquipmentManufacturerForm form={form} />
      <Group position="right" mt="md">
        <Button
          variant="subtle"
          onClick={onCancel}
          color="gray"
          type="button"
          className={classes.control}
        >
          Cancel
        </Button>
        <Button
          color="white"
          type="submit"
          className={classes.control}
          loading={loading}
        >
          Submit
        </Button>
      </Group>
    </form>
  );
}

export function EquipManufacturerDrawer(): React.ReactElement {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [equipManufacturer] = store.use("selectedRecord");
  const onCancel = () => setDrawerOpen(false);

  return (
    <Drawer.Root
      position="right"
      opened={drawerOpen}
      onClose={() => setDrawerOpen(false)}
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>
            Edit Equipment Manufacturer:{" "}
            {equipManufacturer && equipManufacturer.name}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {equipManufacturer && (
            <EditEquipManufacturerForm
              onCancel={onCancel}
              equipManufacturer={equipManufacturer}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
