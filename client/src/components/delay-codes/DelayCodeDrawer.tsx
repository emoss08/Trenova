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
import { useFormStyles } from "@/assets/styles/FormStyles";
import { DelayCode, DelayCodeFormValues as FormValues } from "@/types/dispatch";
import { useDelayCodeStore as store } from "@/stores/DispatchStore";
import { delayCodeSchema } from "@/lib/schemas/DispatchSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { DelayCodeForm } from "@/components/delay-codes/CreateDelayCodeModal";

function EditDelayCodeModalForm({
  delayCode,
  onCancel,
}: {
  delayCode: DelayCode;
  onCancel: () => void;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(delayCodeSchema),
    initialValues: {
      status: delayCode.status,
      code: delayCode.code,
      description: delayCode.description,
      fCarrierOrDriver: delayCode.fCarrierOrDriver,
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<DelayCode>>(
    form,
    notifications,
    {
      method: "PUT",
      path: `/delay_codes/${delayCode.code}/`,
      successMessage: "Delay Code updated successfully.",
      queryKeysToInvalidate: ["delay-code-table-data"],
      additionalInvalidateQueries: ["delayCodes"],
      closeModal: true,
      errorMessage: "Failed to update delay code.",
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
      <div className={classes.div}>
        <DelayCodeForm form={form} />
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
      </div>
    </form>
  );
}

export function DelayCodeDrawer() {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [delayCode] = store.use("selectedRecord");
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
            Edit Delay Code: {delayCode && delayCode.code}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {delayCode && (
            <EditDelayCodeModalForm delayCode={delayCode} onCancel={onCancel} />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
