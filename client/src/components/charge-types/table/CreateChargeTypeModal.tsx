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

import { Button, Group, Modal, SimpleGrid, Skeleton } from "@mantine/core";
import React, { Suspense } from "react";
import { notifications } from "@mantine/notifications";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { chargeTypeTableStore as store } from "@/stores/BillingStores";
import { useFormStyles } from "@/assets/styles/FormStyles";
import {
  ChargeType,
  ChargeTypeFormValues as FormValues,
} from "@/types/billing";
import { chargeTypeSchema } from "@/lib/schemas/BillingSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices } from "@/lib/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";

export function ChargeTypeForm({
  form,
}: {
  form: UseFormReturnType<FormValues>;
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
          description="Status of the Charge Type"
          form={form}
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          className={classes.fields}
          name="name"
          description="Unique name for the Charge Type"
          label="Name"
          placeholder="Name"
          variant="filled"
          withAsterisk
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        className={classes.fields}
        name="description"
        description="Description of the Charge Type"
        label="Description"
        placeholder="Description"
        variant="filled"
      />
    </div>
  );
}

function CreateChargeTypeModalForm() {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(chargeTypeSchema),
    initialValues: {
      status: "A",
      name: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<ChargeType>>(
    form,
    store,
    notifications,
    {
      method: "POST",
      path: "/charge_types/",
      successMessage: "Charge Type created successfully.",
      queryKeysToInvalidate: ["charge-type-table-data"],
      additionalInvalidateQueries: ["chargeTypes"],
      closeModal: true,
      errorMessage: "Failed to create charge type",
    },
    () => setLoading(false),
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <ChargeTypeForm form={form} />
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
    </form>
  );
}

export function CreateChargeTypeModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Charge Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            <CreateChargeTypeModalForm />
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
