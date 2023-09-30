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
import {
  Box,
  Button,
  Drawer,
  Group,
  Select,
  SimpleGrid,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { commodityTableStore as store } from "@/stores/CommodityStore";
import {
  Commodity,
  CommodityFormValues as FormValues,
} from "@/types/commodities";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { yesAndNoChoices } from "@/lib/constants";
import { UnitOfMeasureChoices } from "@/lib/choices";
import { commoditySchema } from "@/lib/schemas/CommoditiesSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import {
  SelectInput,
  ViewSelectInput,
} from "@/components/common/fields/SelectInput";
import { useHazardousMaterial } from "@/hooks/useHazardousMaterial";

type EditCommodityModalFormProps = {
  commodity: Commodity;
  selectHazmatData: TChoiceProps[];
  onCancel: () => void;
  isErrors: boolean;
  isLoading: boolean;
};

function EditCommodityModalForm({
  commodity,
  selectHazmatData,
  isErrors,
  isLoading,
  onCancel,
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

  const mutation = useCustomMutation<FormValues, TableStoreProps<Commodity>>(
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
              isError={isErrors}
              isLoading={isLoading}
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
        </Box>
      </Box>
    </form>
  );
}

type ViewCommodityModalFormProps = {
  commodity: Commodity;
  selectHazmatData: TChoiceProps[];
  onEditClick: () => void;
};

function ViewCommodityModalForm({
  commodity,
  selectHazmatData,
  onEditClick,
}: ViewCommodityModalFormProps) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
        <TextInput
          className={classes.fields}
          value={commodity.name}
          name="name"
          label="Name"
          placeholder="Name"
          readOnly
          variant="filled"
          withAsterisk
        />
        <Textarea
          className={classes.fields}
          name="description"
          label="Description"
          placeholder="Description"
          readOnly
          variant="filled"
          value={commodity.description || ""}
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <TextInput
            className={classes.fields}
            name="minTemp"
            label="Min Temp"
            placeholder="Min Temp"
            readOnly
            variant="filled"
            value={commodity.minTemp || ""}
          />
          <TextInput
            className={classes.fields}
            name="maxTemp"
            label="Max Temp"
            placeholder="Max Temp"
            readOnly
            variant="filled"
            value={commodity.maxTemp || ""}
          />
        </SimpleGrid>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <ViewSelectInput
            className={classes.fields}
            data={selectHazmatData || []}
            placeholder="Hazardous Material"
            label="Hazardous Material"
            variant="filled"
            value={commodity.hazmat || ""}
            readOnly
            clearable
          />
          <Select
            className={classes.fields}
            data={yesAndNoChoices}
            name="isHazmat"
            label="Is Hazmat"
            placeholder="Is Hazmat"
            variant="filled"
            value={commodity.isHazmat || ""}
            readOnly
            withAsterisk
          />
        </SimpleGrid>
        <Select
          className={classes.fields}
          data={UnitOfMeasureChoices}
          name="unitOfMeasure"
          placeholder="Unit of Measure"
          label="Unit of Measure"
          value={commodity.unitOfMeasure || ""}
          readOnly
          variant="filled"
        />
        <Group position="right" mt="md" spacing="xs">
          <Button
            variant="subtle"
            type="submit"
            onClick={onEditClick}
            className={classes.control}
          >
            Edit
          </Button>
          <Button
            color="red"
            type="submit"
            onClick={() => {
              store.set("drawerOpen", false);
              store.set("deleteModalOpen", true);
            }}
            className={classes.control}
          >
            Remove
          </Button>
        </Group>
      </Box>
    </Box>
  );
}

export function CommodityDrawer() {
  const [isEditing, setIsEditing] = React.useState(false);
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [commodity] = store.use("selectedRecord");

  const toggleEditMode = () => {
    setIsEditing((prev) => !prev);
  };

  const { selectHazardousMaterials, isLoading, isError } =
    useHazardousMaterial(drawerOpen);

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
            {isEditing ? "Edit Commodity" : "View Commodity"}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {commodity && isEditing ? (
            <EditCommodityModalForm
              isLoading={isLoading}
              isErrors={isError}
              commodity={commodity}
              selectHazmatData={selectHazardousMaterials}
              onCancel={toggleEditMode}
            />
          ) : (
            commodity && (
              <ViewCommodityModalForm
                commodity={commodity}
                selectHazmatData={selectHazardousMaterials}
                onEditClick={toggleEditMode}
              />
            )
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
