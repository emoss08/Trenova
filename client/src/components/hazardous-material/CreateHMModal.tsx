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

import { Button, Group, Modal, SimpleGrid } from "@mantine/core";
import React from "react";
import { notifications } from "@mantine/notifications";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { hazardousMaterialTableStore as store } from "@/stores/CommodityStore";
import { useFormStyles } from "@/assets/styles/FormStyles";
import {
  HazardousMaterial,
  HazardousMaterialFormValues as FormValues,
} from "@/types/commodities";
import { hazardousMaterialSchema } from "@/lib/validations/CommoditiesSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices } from "@/lib/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { HazardousClassChoices, PackingGroupChoices } from "@/lib/choices";

export function HMForm({ form }: { form: UseFormReturnType<FormValues> }) {
  const { classes } = useFormStyles();

  return (
    <div className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={statusChoices}
          className={classes.fields}
          name="status"
          label="Status"
          placeholder="Status"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          className={classes.fields}
          name="name"
          label="Name"
          placeholder="Name"
          variant="filled"
          withAsterisk
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        className={classes.fields}
        name="description"
        label="Description"
        placeholder="Description"
        variant="filled"
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={HazardousClassChoices}
          className={classes.fields}
          name="hazardClass"
          label="Hazard Class"
          placeholder="Hazard Class"
          variant="filled"
          withAsterisk
        />
        <SelectInput<FormValues>
          form={form}
          data={PackingGroupChoices}
          className={classes.fields}
          name="packingGroup"
          label="Packing Group"
          placeholder="Packing Group"
          variant="filled"
          clearable
        />
      </SimpleGrid>
      <ValidatedTextInput<FormValues>
        form={form}
        className={classes.fields}
        name="ergNumber"
        label="ERG Number"
        placeholder="ERG Number"
        variant="filled"
      />
      <ValidatedTextArea<FormValues>
        form={form}
        className={classes.fields}
        name="properShippingName"
        label="Proper Shipping Name"
        placeholder="Proper Shipping Name"
        variant="filled"
      />
    </div>
  );
}

export function CreateHMModalForm() {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(hazardousMaterialSchema),
    initialValues: {
      status: "A",
      name: "",
      description: "",
      hazardClass: "1.3",
      packingGroup: "I",
      ergNumber: "",
      properShippingName: "",
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    TableStoreProps<HazardousMaterial>
  >(
    form,
    notifications,
    {
      method: "POST",
      path: "/hazardous_materials/",
      successMessage: "Hazardous Material created successfully.",
      queryKeysToInvalidate: ["hazardous-material-table-data"],
      additionalInvalidateQueries: ["hazardousMaterials"],
      closeModal: true,
      errorMessage: "Failed to create hazardous material.",
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
      <HMForm form={form} />
      <Group position="right" mt="md">
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

export function CreateHMModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Hazardous Material</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateHMModalForm />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
