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
import { Button, Group, Modal, SimpleGrid } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { commodityTableStore as store } from "@/stores/CommodityStore";
import {
  Commodity,
  CommodityFormValues as FormValues,
} from "@/types/commodities";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { commoditySchema } from "@/lib/schemas/CommoditiesSchema";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices, yesAndNoChoices } from "@/lib/constants";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { UnitOfMeasureChoices } from "@/lib/choices";
import { useHazardousMaterial } from "@/hooks/useHazardousMaterial";

export function CommodityForm({
  form,
  selectHazmatData,
  isLoading,
  isError,
}: {
  form: UseFormReturnType<FormValues>;
  selectHazmatData: TChoiceProps[];
  isLoading: boolean;
  isError: boolean;
}) {
  const { classes } = useFormStyles();
  return (
    <div className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={statusChoices}
          name="status"
          placeholder="Status"
          label="Status"
          description="Status of the Commodity"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput
          form={form}
          name="name"
          label="Name"
          description="Name of the Commodity"
          placeholder="Name"
          variant="filled"
          withAsterisk
        />
      </SimpleGrid>
      <ValidatedTextArea
        form={form}
        name="description"
        label="Description"
        description="Description of the Commodity"
        placeholder="Description"
        variant="filled"
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <ValidatedTextInput<FormValues>
          form={form}
          name="minTemp"
          label="Min Temp"
          description="Minimum Temperature of the Commodity"
          placeholder="Min. Temp"
          variant="filled"
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="maxTemp"
          description="Maximum Temperature of the Commodity"
          label="Max Temp"
          placeholder="Max. Temp"
          variant="filled"
        />
      </SimpleGrid>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          data={selectHazmatData}
          isLoading={isLoading}
          isError={isError}
          name="hazmat"
          placeholder="Hazardous Material"
          description="Hazardous Material of the Commodity"
          label="Hazardous Material"
          form={form}
          variant="filled"
          clearable
        />
        <SelectInput<FormValues>
          data={yesAndNoChoices}
          name="isHazmat"
          description="Is the Commodity a Hazardous Material"
          label="Is Hazmat"
          placeholder="Is Hazmat"
          form={form}
          variant="filled"
          withAsterisk
        />
      </SimpleGrid>
      <SelectInput<FormValues>
        data={UnitOfMeasureChoices}
        name="unitOfMeasure"
        placeholder="Unit of Measure"
        description="Unit of Measure of the Commodity"
        label="Unit of Measure"
        form={form}
        variant="filled"
      />
    </div>
  );
}

export function CreateCommodityModalForm({
  selectHazmatData,
  isError,
  isLoading,
}: {
  selectHazmatData: TChoiceProps[];
  isError: boolean;
  isLoading: boolean;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(commoditySchema),
    initialValues: {
      status: "A",
      name: "",
      description: "",
      minTemp: undefined,
      maxTemp: undefined,
      setPointTemp: undefined,
      unitOfMeasure: undefined,
      hazmat: "",
      isHazmat: "N",
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<Commodity>>(
    form,
    store,
    notifications,
    {
      method: "POST",
      path: "/commodities/",
      successMessage: "Commodity created successfully.",
      queryKeysToInvalidate: ["commodity-table-data"],
      additionalInvalidateQueries: ["commodities"],
      closeModal: true,
      errorMessage: "Failed to create commodity.",
    },
    () => setLoading(false),
  );

  React.useEffect(() => {
    if (form.values.hazmat) {
      form.setFieldValue("isHazmat", "Y");
    } else {
      form.setFieldValue("isHazmat", "N");
    }
  }, [form.values.hazmat, form]);

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <CommodityForm
        form={form}
        selectHazmatData={selectHazmatData}
        isLoading={isLoading}
        isError={isError}
      />
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

export function CreateCommodityModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const { selectHazardousMaterials, isError, isLoading } =
    useHazardousMaterial(showCreateModal);

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Commodity</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateCommodityModalForm
            selectHazmatData={selectHazardousMaterials}
            isLoading={isLoading}
            isError={isError}
          />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
