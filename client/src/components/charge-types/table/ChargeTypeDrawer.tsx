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

import { Box, Button, Drawer, Group, Textarea, TextInput } from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { chargeTypeTableStore as store } from "@/stores/BillingStores";
import {
  ChargeType,
  ChargeTypeFormValues as FormValues,
} from "@/types/billing";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { chargeTypeSchema } from "@/lib/schemas/BillingSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";

type EditChargeTypeModalFormProps = {
  chargeType: ChargeType;
  onCancel: () => void;
};

function EditChargeTypeModalForm({
  chargeType,
  onCancel,
}: EditChargeTypeModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(chargeTypeSchema),
    initialValues: {
      name: chargeType.name,
      description: chargeType.description,
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<ChargeType>>(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/charge_types/${chargeType.id}/`,
      successMessage: "Charge Type updated successfully.",
      queryKeysToInvalidate: ["charge-type-table-data"],
      additionalInvalidateQueries: ["chargeTypes"],
      closeModal: true,
      errorMessage: "Failed to update charge type.",
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
        <Box>
          <ValidatedTextInput
            form={form}
            className={classes.fields}
            name="name"
            label="Name"
            placeholder="Name"
            variant="filled"
            withAsterisk
          />
          <ValidatedTextArea
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
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
        </Box>
      </Box>
    </form>
  );
}

function ViewChargeTypeModalForm({
  chargeType,
  onEditClick,
}: {
  chargeType: ChargeType;
  onEditClick: () => void;
}) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
        <TextInput
          value={chargeType.name}
          readOnly
          className={classes.fields}
          label="Name"
          variant="filled"
        />
        <Textarea
          value={chargeType.description || ""}
          className={classes.fields}
          label="Description"
          readOnly
          variant="filled"
        />
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
    </Box>
  );
}

export function ChargeTypeDrawer(): React.ReactElement {
  const [isEditing, setIsEditing] = React.useState(false);
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [chargeType] = store.use("selectedRecord");

  const toggleEditMode = () => {
    setIsEditing((prev) => !prev);
  };

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
            {isEditing ? "Edit Charge Type" : "View Charge Type"}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {chargeType && isEditing ? (
            <EditChargeTypeModalForm
              chargeType={chargeType}
              onCancel={toggleEditMode}
            />
          ) : (
            chargeType && (
              <ViewChargeTypeModalForm
                chargeType={chargeType}
                onEditClick={toggleEditMode}
              />
            )
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
