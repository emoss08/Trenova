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
import { commodityTableStore as store } from "@/stores/CommodityStore";
import {
  Commodity,
  CommodityFormValues as FormValues,
} from "@/types/commodities";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { useHazardousMaterial } from "@/hooks/useHazardousMaterial";
import { CommodityForm } from "@/components/commodities/CreateCommodityModal";

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
      status: commodity.status,
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
      <CommodityForm
        form={form}
        selectHazmatData={selectHazmatData}
        isLoading={isLoading}
        isError={isErrors}
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
    </form>
  );
}

export function CommodityDrawer() {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [commodity] = store.use("selectedRecord");
  const onCancel = () => store.set("drawerOpen", false);
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
            Edit Commodity: {commodity && commodity.name}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {commodity && (
            <EditCommodityModalForm
              isLoading={isLoading}
              isErrors={isError}
              commodity={commodity}
              selectHazmatData={selectHazardousMaterials}
              onCancel={onCancel}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
