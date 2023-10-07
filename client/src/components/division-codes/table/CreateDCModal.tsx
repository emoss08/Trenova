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

import { Button, Group, Modal, SimpleGrid } from "@mantine/core";
import React from "react";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { divisionCodeTableStore as store } from "@/stores/AccountingStores";
import {
  DivisionCode,
  DivisionCodeFormValues as FormValues,
} from "@/types/accounting";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { divisionCodeSchema } from "@/lib/schemas/AccountingSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices } from "@/lib/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { useGLAccounts } from "@/hooks/useGLAccounts";

export function DCForm({
  selectGlAccountData,
  isError,
  isLoading,
  form,
}: {
  selectGlAccountData: TChoiceProps[];
  isError: boolean;
  isLoading: boolean;
  form: UseFormReturnType<FormValues>;
}) {
  const { classes } = useFormStyles();

  return (
    <div className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={statusChoices}
          name="status"
          description="Status of the Division Code"
          label="Status"
          placeholder="Status"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="code"
          description="Unique code for the Division Code"
          label="Code"
          placeholder="Code"
          variant="filled"
          withAsterisk
          maxLength={4}
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        name="description"
        description="Description of the Division Code"
        label="Description"
        placeholder="Description"
        variant="filled"
        withAsterisk
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={selectGlAccountData}
          isError={isError}
          isLoading={isLoading}
          name="apAccount"
          description="Associated AP account for Division Code"
          label="AP Account"
          placeholder="AP Account"
          variant="filled"
          clearable
        />
        <SelectInput<FormValues>
          form={form}
          data={selectGlAccountData}
          isError={isError}
          isLoading={isLoading}
          description="Associated cash account for Division Code"
          name="cashAccount"
          label="Cash Account"
          placeholder="Cash Account"
          variant="filled"
          clearable
        />
      </SimpleGrid>
      <SelectInput<FormValues>
        form={form}
        data={selectGlAccountData}
        isError={isError}
        isLoading={isLoading}
        name="expenseAccount"
        description="Associated expense account for Division Code"
        label="Expense Account"
        placeholder="Expense Account"
        variant="filled"
        clearable
      />
    </div>
  );
}

function CreateDCModalForm({
  selectGlAccountData,
  isError,
  isLoading,
}: {
  selectGlAccountData: TChoiceProps[];
  isError: boolean;
  isLoading: boolean;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(divisionCodeSchema),
    initialValues: {
      status: "A",
      code: "",
      description: "",
      apAccount: "",
      cashAccount: "",
      expenseAccount: "",
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<DivisionCode>>(
    form,
    notifications,
    {
      method: "POST",
      path: "/division_codes/",
      successMessage: "Division Code created successfully.",
      queryKeysToInvalidate: ["division-code-table-data"],
      additionalInvalidateQueries: ["divisionCodes"],
      closeModal: true,
      errorMessage: "Failed to create division code.",
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
      <DCForm
        selectGlAccountData={selectGlAccountData}
        isError={isError}
        isLoading={isLoading}
        form={form}
      />
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

export function CreateDCModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const { selectGLAccounts, isError, isLoading } =
    useGLAccounts(showCreateModal);

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Division Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateDCModalForm
            selectGlAccountData={selectGLAccounts}
            isLoading={isLoading}
            isError={isError}
          />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
