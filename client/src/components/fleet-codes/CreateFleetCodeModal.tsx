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

import React, { Suspense } from "react";
import { Button, Group, Modal, SimpleGrid, Skeleton } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { useFleetCodeStore as store } from "@/stores/DispatchStore";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { FleetCode, FleetCodeFormValues as FormValues } from "@/types/dispatch";
import { fleetCodeSchema } from "@/lib/schemas/DispatchSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { TChoiceProps } from "@/types";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { useUsers } from "@/hooks/useUsers";
import { statusChoices } from "@/lib/constants";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

export function FleetCodeForm({
  form,
  users,
  isUsersLoading,
  isUsersError,
  isEdit,
}: {
  form: UseFormReturnType<FormValues>;
  users: ReadonlyArray<TChoiceProps>;
  isUsersLoading: boolean;
  isUsersError: boolean;
  isEdit: boolean;
}) {
  const { classes } = useFormStyles();

  return (
    <div className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          data={statusChoices}
          name="status"
          placeholder="Status"
          label="Status"
          description="Status of the Fleet Code"
          form={form}
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="code"
          label="Code"
          placeholder="Code"
          description="Unique Code of the Fleet Code"
          withAsterisk
          disabled={isEdit}
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        name="description"
        label="Description"
        description="Description of the Fleet Code"
        placeholder="Description"
        withAsterisk
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <ValidatedNumberInput<FormValues>
          form={form}
          name="revenueGoal"
          label="Revenue Goal"
          placeholder="Revenue Goal"
          precision={2}
          withAsterisk
          description="Revenue goal for Fleet Code"
        />
        <ValidatedNumberInput<FormValues>
          form={form}
          name="deadheadGoal"
          label="Deadhead Goal"
          placeholder="Deadhead Goal"
          precision={2}
          withAsterisk
          description="Deadhead goal for Fleet Code"
        />
        <ValidatedNumberInput<FormValues>
          form={form}
          name="mileageGoal"
          label="Mileage Goal"
          placeholder="Mileage Goal"
          precision={2}
          withAsterisk
          description="Mileage goal for Fleet Code"
        />
        <SelectInput<FormValues>
          form={form}
          name="manager"
          label="Manager"
          placeholder="Manager"
          description="Manger of the Fleet Code"
          data={users}
          isLoading={isUsersLoading}
          isError={isUsersError}
        />
      </SimpleGrid>
    </div>
  );
}

export function CreateFleetCodeModalForm({
  users,
  isUsersLoading,
  isUsersError,
}: {
  users: ReadonlyArray<TChoiceProps>;
  isUsersLoading: boolean;
  isUsersError: boolean;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(fleetCodeSchema),
    initialValues: {
      status: "A",
      code: "",
      description: "",
      revenueGoal: 0.0,
      deadheadGoal: 0.0,
      mileageGoal: 0.0,
      manager: "",
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<FleetCode>>(
    form,
    notifications,
    {
      method: "POST",
      path: "/fleet_codes/",
      successMessage: "Fleet Code created successfully.",
      queryKeysToInvalidate: ["fleet-code-table-data"],
      additionalInvalidateQueries: ["fleetCodes"],
      closeModal: true,
      errorMessage: "Failed to create fleet code.",
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
        isUsersError={isUsersError}
        isUsersLoading={isUsersLoading}
        users={users}
        isEdit={false}
      />
      <Group position="right" mt="md">
        <Button type="submit" className={classes.control} loading={loading}>
          Submit
        </Button>
      </Group>
    </form>
  );
}

export function CreateFleetCodeModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const {
    selectUsersData,
    isLoading: isUsersLoading,
    isError: isUsersError,
  } = useUsers(showCreateModal);

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Fleet Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            <CreateFleetCodeModalForm
              users={selectUsersData}
              isUsersLoading={isUsersLoading}
              isUsersError={isUsersError}
            />
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
