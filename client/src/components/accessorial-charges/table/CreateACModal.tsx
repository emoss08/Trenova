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

import { Box, Button, Group, Modal, SimpleGrid } from "@mantine/core";
import React from "react";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { accessorialChargeTableStore } from "@/stores/BillingStores";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { AccessorialChargeFormValues } from "@/types/billing";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { accessorialChargeSchema } from "@/helpers/schemas/BillingSchema";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { fuelMethodChoices } from "@/utils/apps/billing";
import { SwitchInput } from "@/components/common/fields/SwitchInput";

export function CreateACModalForm() {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: AccessorialChargeFormValues) =>
      axios.post("/accessorial_charges/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["accessorial-charges-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Accessorial Charge created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            accessorialChargeTableStore.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<AccessorialChargeFormValues>({
    validate: yupResolver(accessorialChargeSchema),
    initialValues: {
      code: "",
      description: "",
      isDetention: false,
      chargeAmount: undefined,
      method: "D",
    },
  });

  const submitForm = (values: AccessorialChargeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <Box>
          <ValidatedTextInput<AccessorialChargeFormValues>
            form={form}
            className={classes.fields}
            name="code"
            label="Code"
            description="Code for the accessorial charge"
            placeholder="Code"
            variant="filled"
            withAsterisk
          />
          <ValidatedTextArea<AccessorialChargeFormValues>
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            description="Description of the accessorial charge"
            placeholder="Description"
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <ValidatedTextInput<AccessorialChargeFormValues>
              form={form}
              className={classes.fields}
              name="chargeAmount"
              label="Charge Amount"
              placeholder="Charge Amount"
              description="Charge amount for the accessorial charge"
              variant="filled"
              withAsterisk
            />
            <SelectInput<AccessorialChargeFormValues>
              form={form}
              data={fuelMethodChoices}
              className={classes.fields}
              name="method"
              label="Fuel Method"
              description="Method for calculating the accessorial charge"
              placeholder="Fuel Method"
              variant="filled"
              withAsterisk
            />
            <SwitchInput<AccessorialChargeFormValues>
              form={form}
              className={classes.fields}
              name="isDetention"
              label="Detention"
              description="Is this a detention charge?"
              placeholder="Detention"
              variant="filled"
            />
          </SimpleGrid>
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

export function CreateACModal() {
  const [showCreateModal, setShowCreateModal] =
    accessorialChargeTableStore.use("createModalOpen");

  if (!showCreateModal) return null;

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      size="30%"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Accessorial Charge</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateACModalForm />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
