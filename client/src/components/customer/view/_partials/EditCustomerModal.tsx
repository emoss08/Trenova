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
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { customerStore as store } from "@/stores/CustomerStore";
import { Customer, CustomerFormValues } from "@/types/customer";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/lib/AxiosConfig";
import { customerSchema } from "@/lib/schemas/CustomerSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices, yesAndNoChoices } from "@/lib/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { CityAutoCompleteField } from "@/components/common/fields/CityAutoCompleteField";
import { StateSelect } from "@/components/common/fields/StateSelect";

EditCustomerModalForm.defaultProps = {
  customer: null,
};

EditCustomerModal.defaultProps = {
  customer: null,
};

type EditCustomerModalProps = {
  customer?: Customer | null;
};

function EditCustomerModalForm({ customer }: EditCustomerModalProps) {
  const { classes } = useFormStyles();
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: CustomerFormValues) =>
      axios.put(`/customers/${customer?.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["customer", customer?.id],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Customer updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            setShowEditModal(false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: any) => {
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
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<CustomerFormValues>({
    validate: yupResolver(customerSchema),
    initialValues: {
      name: customer?.name || "",
      status: customer?.status || "A",
      code: customer?.code || "",
      city: customer?.city || "",
      state: customer?.state || "",
      addressLine1: customer?.addressLine1 || "",
      addressLine2: customer?.addressLine2 || "",
      zipCode: customer?.zipCode || "",
      hasCustomerPortal: customer?.hasCustomerPortal || "",
      autoMarkReadyToBill: customer?.autoMarkReadyToBill || "",
    },
  });

  if (!showEditModal) return null;

  const submitForm = (values: CustomerFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <SelectInput<CustomerFormValues>
          className={classes.fields}
          data={statusChoices}
          name="status"
          placeholder="Status"
          label="Status"
          description="Status of the customer"
          form={form}
          variant="filled"
          withAsterisk
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <ValidatedTextInput<CustomerFormValues>
            form={form}
            className={classes.fields}
            name="code"
            label="Code"
            placeholder="Code"
            description="Unique code for the customer"
            variant="filled"
            withAsterisk
            readOnly
          />
          <ValidatedTextInput<CustomerFormValues>
            form={form}
            className={classes.fields}
            name="name"
            description="Name of the customer"
            label="Name"
            placeholder="Name"
            variant="filled"
            withAsterisk
          />
        </SimpleGrid>
        <ValidatedTextInput<CustomerFormValues>
          form={form}
          className={classes.fields}
          name="addressLine1"
          description="Address Line 1 of the customer"
          label="Address Line 1"
          placeholder="Address Line 1"
          variant="filled"
        />
        <ValidatedTextInput<CustomerFormValues>
          form={form}
          className={classes.fields}
          description="Address Line 2 of the customer"
          name="addressLine2"
          label="Address Line 2"
          placeholder="Address Line 2"
          variant="filled"
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <CityAutoCompleteField<CustomerFormValues>
            form={form}
            stateSelection={form.values.city || ""}
            className={classes.fields}
            name="city"
            description="City of the customer"
            label="City"
            placeholder="City"
            variant="filled"
          />
          <StateSelect<CustomerFormValues>
            label="State"
            className={classes.fields}
            placeholder="State"
            variant="filled"
            description="State of the customer"
            searchable
            form={form}
            name="state"
          />
        </SimpleGrid>
        <ValidatedTextInput<CustomerFormValues>
          form={form}
          className={classes.fields}
          name="zipCode"
          label="Zip Code"
          description="Zip Code of the customer"
          placeholder="Zip Code"
          variant="filled"
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<CustomerFormValues>
            className={classes.fields}
            data={yesAndNoChoices}
            name="hasCustomerPortal"
            placeholder="Has Customer Portal"
            label="Has Customer Portal"
            description="Customer has Customer Portal?"
            form={form}
            variant="filled"
            withAsterisk
          />
          <SelectInput<CustomerFormValues>
            className={classes.fields}
            data={yesAndNoChoices}
            name="autoMarkReadyToBill"
            placeholder="Auto Mark Ready to Bill"
            label="Auto Mark Ready to Bill"
            description="Auto Mark Ready to Bill?"
            form={form}
            variant="filled"
            withAsterisk
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

export function EditCustomerModal({
  customer,
}: EditCustomerModalProps): React.ReactElement | null {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");

  if (!showEditModal) return null;

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Customer</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {customer && <EditCustomerModalForm customer={customer} />}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
