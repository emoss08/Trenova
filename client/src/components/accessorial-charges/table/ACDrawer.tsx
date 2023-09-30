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

import {
  Box,
  Button,
  Drawer,
  Group,
  Select,
  SimpleGrid,
  Switch,
  Textarea,
  TextInput,
} from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { accessorialChargeTableStore as store } from "@/stores/BillingStores";
import {
  AccessorialCharge,
  AccessorialChargeFormValues as FormValues,
} from "@/types/billing";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { fuelMethodChoices } from "@/utils/apps/billing";
import { accessorialChargeSchema as Schema } from "@/lib/schemas/BillingSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { SwitchInput } from "@/components/common/fields/SwitchInput";

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
      <Box className={classes.div}>
        <ValidatedTextInput<FormValues>
          form={form}
          className={classes.fields}
          name="code"
          label="Code"
          description="Code for the accessorial charge."
          placeholder="Code"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextArea<FormValues>
          form={form}
          className={classes.fields}
          name="description"
          label="Description"
          description="Description of the accessorial charge."
          placeholder="Description"
          variant="filled"
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <ValidatedTextInput<FormValues>
            form={form}
            className={classes.fields}
            name="chargeAmount"
            label="Charge Amount"
            placeholder="Charge Amount"
            description="Charge amount for the accessorial charge."
            variant="filled"
            withAsterisk
          />
          <SelectInput<FormValues>
            form={form}
            data={fuelMethodChoices}
            className={classes.fields}
            name="method"
            label="Fuel Method"
            description="Method for calculating the other charge."
            placeholder="Fuel Method"
            variant="filled"
          />
          <SwitchInput<FormValues>
            form={form}
            className={classes.fields}
            name="isDetention"
            label="Detention"
            description="Is detention charge?"
            placeholder="Detention"
            variant="filled"
          />
        </SimpleGrid>
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
    </form>
  );
}

function ViewACModalForm({
  accessorialCharge,
  onEditClick,
}: {
  accessorialCharge: AccessorialCharge;
  onEditClick: () => void;
}) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <TextInput
        className={classes.fields}
        name="code"
        label="Code"
        description="Code for the accessorial charge."
        placeholder="Code"
        variant="filled"
        readOnly
        value={accessorialCharge.code}
      />
      <Textarea
        className={classes.fields}
        name="description"
        label="Description"
        description="Description of the accessorial charge."
        placeholder="Description"
        variant="filled"
        readOnly
        value={accessorialCharge.description || ""}
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <TextInput
          className={classes.fields}
          name="charge_amount"
          label="Charge Amount"
          placeholder="Charge Amount"
          description="Charge amount for the accessorial charge."
          variant="filled"
          readOnly
          value={accessorialCharge.chargeAmount}
        />
        <Select
          data={fuelMethodChoices}
          className={classes.fields}
          name="method"
          label="Fuel Method"
          description="Method for calculating the other charge."
          placeholder="Fuel Method"
          variant="filled"
          readOnly
          value={accessorialCharge.method}
        />
        <Switch
          className={classes.fields}
          name="is_detention"
          label="Detention"
          description="Is detention charge?"
          placeholder="Detention"
          variant="filled"
          readOnly
          checked={accessorialCharge.isDetention}
        />
      </SimpleGrid>
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
  );
}

export function ACDrawer(): React.ReactElement {
  const [isEditing, setIsEditing] = React.useState(false);
  const [showViewModal, setShowViewModal] = store.use("drawerOpen");
  const [accessorialCharge] = store.use("selectedRecord");

  const toggleEditMode = () => {
    setIsEditing((prev) => !prev);
  };
  return (
    <Drawer.Root
      position="right"
      opened={showViewModal}
      onClose={() => setShowViewModal(false)}
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>
            {isEditing ? "Edit Accessorial Charge" : "View Accessorial Charge"}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {accessorialCharge && isEditing ? (
            <EditACModalForm
              accessorialCharge={accessorialCharge}
              onCancel={toggleEditMode}
            />
          ) : (
            accessorialCharge && (
              <ViewACModalForm
                accessorialCharge={accessorialCharge}
                onEditClick={toggleEditMode}
              />
            )
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
