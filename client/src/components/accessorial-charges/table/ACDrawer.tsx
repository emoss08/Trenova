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
import { accessorialChargeTableStore as store } from "@/stores/BillingStores";
import {
  AccessorialCharge,
  AccessorialChargeFormValues as FormValues,
} from "@/types/billing";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { accessorialChargeSchema as Schema } from "@/lib/schemas/BillingSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { ACForm } from "@/components/accessorial-charges/table/CreateACModal";

function EditACModalForm({
  accessorialCharge,
  onCancel,
}: {
  accessorialCharge: AccessorialCharge;
  onCancel: () => void;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(Schema),
    initialValues: {
      status: accessorialCharge.status,
      code: accessorialCharge.code,
      description: accessorialCharge.description || "",
      isDetention: accessorialCharge.isDetention,
      chargeAmount: accessorialCharge.chargeAmount,
      method: accessorialCharge.method,
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    TableStoreProps<AccessorialCharge>
  >(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/accessorial_charges/${accessorialCharge.id}/`,
      successMessage: "Accessorial Charge updated successfully.",
      queryKeysToInvalidate: ["accessorial-charges-table-data"],
      additionalInvalidateQueries: ["accessorialCharges"],
      closeModal: true,
      errorMessage: "Failed to update accessorial charge.",
    },
    () => setLoading(false),
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <ACForm form={form} />
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

export function ACDrawer(): React.ReactElement {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [accessorialCharge] = store.use("selectedRecord");
  const onCancel = () => store.set("drawerOpen", false);

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
            Edit Accessorial Charge:{" "}
            {accessorialCharge && accessorialCharge.code}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {accessorialCharge && (
            <EditACModalForm
              accessorialCharge={accessorialCharge}
              onCancel={onCancel}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
