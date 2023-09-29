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
import { Box, Button, Group, Modal, SimpleGrid } from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { commodityTableStore as store } from "@/stores/CommodityStore";
import { getHazardousMaterials } from "@/services/CommodityRequestService";
import {
  Commodity,
  CommodityFormValues as FormValues,
  HazardousMaterial,
} from "@/types/commodities";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { commoditySchema } from "@/lib/schemas/CommoditiesSchema";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { yesAndNoChoices } from "@/lib/constants";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { UnitOfMeasureChoices } from "@/lib/choices";

type EditCommodityModalFormProps = {
  commodity: Commodity;
  selectHazmatData: TChoiceProps[];
};

function EditCommodityModalForm({
  commodity,
  selectHazmatData,
}: EditCommodityModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(commoditySchema),
    initialValues: {
      name: commodity.name,
      description: commodity.description,
      minTemp: commodity.minTemp,
      maxTemp: commodity.maxTemp,
      setPointTemp: commodity.setPointTemp,
      unitOfMeasure: commodity.unitOfMeasure,
      hazmat: commodity.hazmat,
      isHazmat: commodity.isHazmat,
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    Omit<TableStoreProps<Commodity>, "drawerOpen">
  >(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/commodities/${commodity.id}/`,
      successMessage: "Commodity updated successfully.",
      queryKeysToInvalidate: ["commodity-table-data"],
      additionalInvalidateQueries: ["commodities"],
      closeModal: true,
      errorMessage: "Failed to update commodity.",
    },
    () => setLoading(false),
  );

  // Set is_hazmat value based on hazmat value
  React.useEffect(() => {
    if (form.values.hazmat) {
      form.setFieldValue("isHazmat", "Y");
    } else {
      form.setFieldValue("isHazmat", "N");
    }
  });

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <Box>
          <ValidatedTextInput<FormValues>
            form={form}
            className={classes.fields}
            name="name"
            label="Name"
            placeholder="Name"
            variant="filled"
            withAsterisk
          />
          <ValidatedTextArea<FormValues>
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <ValidatedTextInput<FormValues>
              form={form}
              className={classes.fields}
              name="minTemp"
              label="Min Temp"
              placeholder="Min Temp"
              variant="filled"
            />
            <ValidatedTextInput<FormValues>
              form={form}
              className={classes.fields}
              name="maxTemp"
              label="Max Temp"
              placeholder="Max Temp"
              variant="filled"
            />
          </SimpleGrid>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<FormValues>
              className={classes.fields}
              data={selectHazmatData || []}
              name="hazmat"
              placeholder="Hazardous Material"
              label="Hazardous Material"
              form={form}
              variant="filled"
              clearable
            />
            <SelectInput<FormValues>
              className={classes.fields}
              data={yesAndNoChoices}
              name="isHazmat"
              label="Is Hazmat"
              placeholder="Is Hazmat"
              form={form}
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <SelectInput<FormValues>
            className={classes.fields}
            data={UnitOfMeasureChoices}
            name="unitOfMeasure"
            placeholder="Unit of Measure"
            label="Unit of Measure"
            form={form}
            variant="filled"
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
        </Box>
      </Box>
    </form>
  );
}

export function EditCommodityModal() {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [commodity] = store.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: hazmatData } = useQuery({
    queryKey: "hazmat-data",
    queryFn: () => getHazardousMaterials(),
    enabled: showEditModal,
    initialData: () => queryClient.getQueryData("hazmat-data"),
    staleTime: Infinity,
  });

  const selectHazmatData =
    hazmatData?.map((hazardousMaterial: HazardousMaterial) => ({
      value: hazardousMaterial.id,
      label: hazardousMaterial.name,
    })) || [];

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Commodity</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {commodity && (
            <EditCommodityModalForm
              commodity={commodity}
              selectHazmatData={selectHazmatData}
            />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
