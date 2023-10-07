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
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { useFormStyles } from "@/assets/styles/FormStyles";
import {
  HazardousMaterial,
  HazardousMaterialFormValues as FormValues,
} from "@/types/commodities";
import { hazardousMaterialTableStore as store } from "@/stores/CommodityStore";
import { hazardousMaterialSchema } from "@/lib/schemas/CommoditiesSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { HMForm } from "@/components/hazardous-material/CreateHMModal";

function EditHMModalForm({
  hazardousMaterial,
  onCancel,
}: {
  hazardousMaterial: HazardousMaterial;
  onCancel: () => void;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(hazardousMaterialSchema),
    initialValues: {
      status: hazardousMaterial.status,
      name: hazardousMaterial.name,
      description: hazardousMaterial.description,
      hazardClass: hazardousMaterial.hazardClass,
      packingGroup: hazardousMaterial.packingGroup,
      ergNumber: hazardousMaterial.ergNumber,
      properShippingName: hazardousMaterial.properShippingName,
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    TableStoreProps<HazardousMaterial>
  >(
    form,
    notifications,
    {
      method: "PUT",
      path: `/hazardous_materials/${hazardousMaterial.id}/`,
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

export function HMDrawer(): React.ReactElement {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [hazardousMaterial] = store.use("selectedRecord");
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
            Edit Hazardous Material:{" "}
            {hazardousMaterial && hazardousMaterial.name}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {hazardousMaterial && (
            <EditHMModalForm
              hazardousMaterial={hazardousMaterial}
              onCancel={onCancel}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
