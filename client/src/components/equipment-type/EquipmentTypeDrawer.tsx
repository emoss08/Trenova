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
import { Button, Drawer, Group } from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { useEquipTypeTableStore as store } from "@/stores/EquipmentStore";
import {
  EquipmentType,
  EquipmentTypeFormValues as FormValues,
} from "@/types/equipment";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { equipmentTypeSchema } from "@/lib/validations/EquipmentSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  EquipmentTypeDetailForm,
  EquipmentTypeForm,
} from "@/components/equipment-type/CreateEquipmentTypeModal";
import { TableStoreProps } from "@/types/tables";

function EditEquipmentTypeForm({
  equipType,
  onCancel,
}: {
  equipType: EquipmentType;
  onCancel: () => void;
}) {
  const [loading, setLoading] = React.useState<boolean>(false);
  const { classes } = useFormStyles();

  const form = useForm<FormValues>({
    validate: yupResolver(equipmentTypeSchema),
    initialValues: {
      status: equipType.status,
      name: equipType.name,
      description: equipType.description,
      costPerMile: equipType.costPerMile,
      equipmentTypeDetails: {
        equipmentClass: equipType.equipmentTypeDetails.equipmentClass,
        exemptFromTolls: equipType.equipmentTypeDetails.exemptFromTolls,
        fixedCost: equipType.equipmentTypeDetails.fixedCost,
        height: equipType.equipmentTypeDetails.height,
        length: equipType.equipmentTypeDetails.length,
        idlingFuelUsage: equipType.equipmentTypeDetails.idlingFuelUsage,
        weight: equipType.equipmentTypeDetails.weight,
        variableCost: equipType.equipmentTypeDetails.variableCost,
        width: equipType.equipmentTypeDetails.width,
      },
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    TableStoreProps<EquipmentType>
  >(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/equipment_types/${equipType.id}/`,
      successMessage: "Equipment type updated successfully.",
      queryKeysToInvalidate: ["equipment-type-table-data"],
      additionalInvalidateQueries: ["equipmentTypes"],
      closeModal: true,
      errorMessage: "Failed to update equipment type.",
    },
    () => setLoading(false),
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <EquipmentTypeForm form={form} />
      <EquipmentTypeDetailForm form={form} />
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

export function EquipmentTypeDrawer() {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [equipType] = store.use("selectedRecord");
  const onCancel = () => setDrawerOpen(false);

  return (
    <Drawer.Root
      position="right"
      opened={drawerOpen}
      onClose={() => setDrawerOpen(false)}
      size="lg"
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>
            Edit Equipment Type: {equipType && equipType.name}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {equipType && (
            <EditEquipmentTypeForm equipType={equipType} onCancel={onCancel} />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
