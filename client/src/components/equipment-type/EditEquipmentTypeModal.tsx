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
import { Button, Group, Modal, Skeleton } from "@mantine/core";
import React, { Suspense } from "react";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { useEquipTypeTableStore as store } from "@/stores/EquipmentStore";
import {
  EquipmentType,
  EquipmentTypeFormValues as FormValues,
} from "@/types/equipment";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { equipmentTypeSchema } from "@/helpers/schemas/EquipmentSchema";
import { usePutMutation } from "@/hooks/useCustomMutation";
import {
  EquipmentTypeDetailForm,
  EquipmentTypeForm,
} from "@/components/equipment-type/CreateEquipmentTypeModal";

function ModalBody({ equipType }: { equipType: EquipmentType }) {
  const [loading, setLoading] = React.useState<boolean>(false);
  const { classes } = useFormStyles();

  const form = useForm<FormValues>({
    validate: yupResolver(equipmentTypeSchema),
    initialValues: {
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

  const mutation = usePutMutation<FormValues>(
    form,
    store,
    notifications,
    {
      path: `/equipment_types/${equipType.id}/`,
      successMessage: "Equipment type updated successfully.",
      queryKeysToInvalidate: ["equipment-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to update equipment type.",
      notificationId: "update-equipment-type",
      validationDetail:
        "Equipment Type with this Name and Organization already exists.",
      validationFieldName: "name",
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
        <Button type="submit" className={classes.control} loading={loading}>
          Submit
        </Button>
      </Group>
    </form>
  );
}

export function EditEquipmentTypeModal() {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [equipType] = store.use("selectedRecord");

  if (!showEditModal) {
    return null;
  }

  return (
    <Modal.Root
      opened={showEditModal}
      onClose={() => setShowEditModal(false)}
      size="lg"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Equipment Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={600} />}>
            {equipType && <ModalBody equipType={equipType} />}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
