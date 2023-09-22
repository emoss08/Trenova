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
import { Box, Button, Group, Modal, SimpleGrid, Skeleton } from "@mantine/core";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { useFleetCodeStore as store } from "@/stores/DispatchStore";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { FleetCodeFormValues } from "@/types/dispatch";
import { fleetCodeSchema } from "@/helpers/schemas/DispatchSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { TChoiceProps } from "@/types";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { useUsers } from "@/hooks/useUsers";
import { yesAndNoChoicesBoolean } from "@/helpers/constants";

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
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: FleetCodeFormValues) => axios.post("/fleet_codes/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["fleet-code-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Fleet Code created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            store.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "nonFieldErrors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            } else if (
              e.attr === "All" &&
              e.detail ===
                "Fleet Code with this Code and Organization already exists."
            ) {
              form.setFieldError("code", e.detail);
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<FleetCodeFormValues>({
    validate: yupResolver(fleetCodeSchema),
    initialValues: {
      code: "",
      description: "",
      isActive: true,
      revenueGoal: 0.0,
      deadheadGoal: 0.0,
      mileageGoal: 0.0,
      manager: "",
    },
  });

  const submitForm = (values: FleetCodeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<FleetCodeFormValues>
            form={form}
            name="isActive"
            label="Is Active"
            description="Is this Fleet Code active?"
            withAsterisk
            data={yesAndNoChoicesBoolean}
          />
          <ValidatedTextInput<FleetCodeFormValues>
            form={form}
            name="code"
            label="Code"
            placeholder="Code"
            description="Unique Code of the Fleet Code"
            withAsterisk
          />
        </SimpleGrid>
        <ValidatedTextArea<FleetCodeFormValues>
          form={form}
          name="description"
          label="Description"
          description="Description of the Fleet Code"
          placeholder="Description"
          withAsterisk
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <ValidatedNumberInput<FleetCodeFormValues>
            form={form}
            name="revenueGoal"
            label="Revenue Goal"
            placeholder="Revenue Goal"
            precision={2}
            withAsterisk
            description="Revenue goal for Fleet Code"
          />
          <ValidatedNumberInput<FleetCodeFormValues>
            form={form}
            name="deadheadGoal"
            label="Deadhead Goal"
            placeholder="Deadhead Goal"
            precision={2}
            withAsterisk
            description="Deadhead goal for Fleet Code"
          />
          <ValidatedNumberInput<FleetCodeFormValues>
            form={form}
            name="mileageGoal"
            label="Mileage Goal"
            placeholder="Mileage Goal"
            precision={2}
            withAsterisk
            description="Mileage goal for Fleet Code"
          />
          <SelectInput<FleetCodeFormValues>
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
        <Group position="right" mt="md">
          <Button type="submit" className={classes.control} loading={loading}>
            Submit
          </Button>
        </Group>
      </Box>
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

  if (!showCreateModal) return null;

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
