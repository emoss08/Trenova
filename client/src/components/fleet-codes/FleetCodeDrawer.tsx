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
import { useFleetCodeStore as store } from "@/stores/DispatchStore";
import { fleetCodeSchema } from "@/lib/validations/DispatchSchema";
import { FleetCode, FleetCodeFormValues as FormValues } from "@/types/dispatch";
import { useUsers } from "@/hooks/useUsers";
import { TChoiceProps } from "@/types";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { FleetCodeForm } from "./CreateFleetCodeModal";

function EditFleetCodeModalForm({
  fleetCode,
  users,
  isUsersLoading,
  isUsersError,
  onCancel,
}: {
  fleetCode: FleetCode;
  users: ReadonlyArray<TChoiceProps>;
  isUsersLoading: boolean;
  isUsersError: boolean;
  onCancel: () => void;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(fleetCodeSchema),
    initialValues: {
      status: fleetCode.status,
      code: fleetCode.code,
      description: fleetCode.description,
      revenueGoal: Number(fleetCode.revenueGoal),
      deadheadGoal: Number(fleetCode.deadheadGoal),
      mileageGoal: Number(fleetCode.mileageGoal),
      manager: fleetCode.manager,
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<FleetCode>>(
    form,
    notifications,
    {
      method: "PUT",
      path: `/fleet_codes/${fleetCode.code}/`,
      successMessage: "Fleet Code updated successfully.",
      queryKeysToInvalidate: ["fleet-code-table-data"],
      additionalInvalidateQueries: ["fleetCodes"],
      closeModal: true,
      errorMessage: "Failed to update fleet code.",
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
      <FleetCodeForm
        form={form}
        users={users}
        isUsersLoading={isUsersLoading}
        isUsersError={isUsersError}
        isEdit
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

export function FleetCodeDrawer() {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [fleetCode] = store.use("selectedRecord");
  const onCancel = () => setDrawerOpen(false);

  const {
    selectUsersData,
    isLoading: isUsersLoading,
    isError: isUsersError,
  } = useUsers(drawerOpen);

  return (
    <Drawer.Root
      position="right"
      opened={drawerOpen}
      onClose={() => setDrawerOpen(false)}
      size="lg"
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>
            Edit Fleet Code: {fleetCode && fleetCode.code}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {fleetCode && (
            <EditFleetCodeModalForm
              fleetCode={fleetCode}
              users={selectUsersData}
              isUsersLoading={isUsersLoading}
              isUsersError={isUsersError}
              onCancel={onCancel}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
