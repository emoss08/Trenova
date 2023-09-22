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

import { notifications } from "@mantine/notifications";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import React, { Suspense } from "react";
import {
  Box,
  Button,
  Divider,
  Group,
  Modal,
  SimpleGrid,
  Skeleton,
  Text,
} from "@mantine/core";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { EquipmentTypeFormValues as FormValues } from "@/types/equipment";
import { equipmentTypeSchema } from "@/helpers/schemas/EquipmentSchema";
import { useEquipTypeTableStore as store } from "@/stores/EquipmentStore";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { equipmentClassChoices } from "@/helpers/choices";
import { yesAndNoChoicesBoolean } from "@/helpers/constants";

export function EquipmentTypeDetailForm({
  form,
}: {
  form: UseFormReturnType<FormValues>;
}): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <div
        style={{
          textAlign: "center",
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          flexDirection: "column",
        }}
      >
        <Text fz="lg" className={classes.text}>
          Equipment Type Details
        </Text>
      </div>
      <Divider my={5} variant="dashed" />
      <SimpleGrid cols={3} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={equipmentClassChoices}
          name="equipmentTypeDetails.equipmentClass"
          label="Equipment Class"
          placeholder="Equipment Class"
          description="Equipment Class associated with the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.fixedCost"
          label="Fixed Cost"
          placeholder="Fixed Cost"
          description="Fixed cost to operate the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.variableCost"
          label="Variable Cost"
          placeholder="Variable Cost"
          description="Variable cost to operate the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.height"
          label="Height"
          placeholder="Height"
          description="Height of the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.weight"
          label="Weight"
          placeholder="Weight"
          description="Weight of the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.length"
          label="Length"
          placeholder="Length"
          description="Length of the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.width"
          label="Width"
          placeholder="Width"
          description="Current Width of the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="equipmentTypeDetails.idlingFuelUsage"
          label="Idling Fuel Usage"
          placeholder="Idling Fuel Usage"
          description="Idling fuel usage of the equipment type."
          withAsterisk
        />
        <SelectInput<FormValues>
          form={form}
          data={yesAndNoChoicesBoolean}
          name="equipmentTypeDetails.exemptFromTolls"
          label="Exempt From Tolls"
          placeholder="Exempt From Tolls"
          description="Exempt from tolls of the equipment type."
          withAsterisk
        />
      </SimpleGrid>
    </Box>
  );
}

export function EquipmentTypeForm({
  form,
}: {
  form: UseFormReturnType<FormValues>;
}) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
        <ValidatedTextInput<FormValues>
          form={form}
          name="name"
          label="Name"
          placeholder="Name"
          description="Unique name for the equipment type."
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="costPerMile"
          label="Cost Per Mile"
          placeholder="Cost Per Mile"
          description="Cost per mile to operate the equipment type."
          withAsterisk
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        name="description"
        label="Description"
        description="Description of the equipment type."
        placeholder="Description"
      />
    </Box>
  );
}

function ModalBody() {
  const [loading, setLoading] = React.useState<boolean>(false);
  const { classes } = useFormStyles();

  const form = useForm<FormValues>({
    validate: yupResolver(equipmentTypeSchema),
    initialValues: {
      name: "",
      description: "",
      costPerMile: "0.00",
      equipmentTypeDetails: {
        equipmentClass: "UNDEFINED",
        exemptFromTolls: false,
        fixedCost: "0.00",
        height: "0.00",
        length: "0.00",
        idlingFuelUsage: "0.00",
        weight: "0.00",
        variableCost: "0.00",
        width: "0.00",
      },
    },
  });

  const mutation = useCustomMutation<FormValues>(
    form,
    store,
    notifications,
    {
      path: "/equipment_types/",
      successMessage: "Equipment type created successfully.",
      queryKeysToInvalidate: ["equipment-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to create equipment type.",
      notificationId: "create-equipment-type",
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

export function CreateEquipmentTypeModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");

  if (!showCreateModal) {
    return null;
  }

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      size="lg"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Equipment Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={600} />}>
            <ModalBody />
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
