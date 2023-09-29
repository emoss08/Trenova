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
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
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

type EditDCModalFormProps = {
  divisionCode: DivisionCode;
  selectGlAccountData: TChoiceProps[];
  isGLAccountsLoading: boolean;
  isGLAccountsError: boolean;
};

export function EditDCModalForm({
  divisionCode,
  selectGlAccountData,
  isGLAccountsError,
  isGLAccountsLoading,
}: EditDCModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(divisionCodeSchema),
    initialValues: {
      status: divisionCode.status,
      code: divisionCode.code,
      description: divisionCode.description,
      apAccount: divisionCode.apAccount || "",
      cashAccount: divisionCode.cashAccount || "",
      expenseAccount: divisionCode.expenseAccount || "",
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    Omit<TableStoreProps<DivisionCode>, "drawerOpen">
  >(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/division_codes/${divisionCode.id}/`,
      successMessage: "Division Code updated successfully.",
      queryKeysToInvalidate: ["division-code-table-data"],
      additionalInvalidateQueries: ["divisionCodes"],
      closeModal: true,
      errorMessage: "Failed to updated division code.",
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
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<FormValues>
              form={form}
              data={statusChoices}
              className={classes.fields}
              name="status"
              label="Status"
              placeholder="Status"
              variant="filled"
            />
            <ValidatedTextInput<FormValues>
              form={form}
              className={classes.fields}
              name="code"
              label="Code"
              placeholder="Code"
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <ValidatedTextArea<FormValues>
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
            withAsterisk
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<FormValues>
              form={form}
              data={selectGlAccountData}
              isLoading={isGLAccountsLoading}
              isError={isGLAccountsError}
              className={classes.fields}
              name="apAccount"
              label="AP Account"
              placeholder="AP Account"
              variant="filled"
              clearable
            />
            <SelectInput<FormValues>
              form={form}
              data={selectGlAccountData}
              isLoading={isGLAccountsLoading}
              isError={isGLAccountsError}
              className={classes.fields}
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
            isLoading={isGLAccountsLoading}
            isError={isGLAccountsError}
            className={classes.fields}
            name="expenseAccount"
            label="Expense Account"
            placeholder="Expense Account"
            variant="filled"
            clearable
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
        </Box>
      </Box>
    </form>
  );
}

export function EditDCModal(): React.ReactElement {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [divisionCode] = store.use("selectedRecord");
  const { selectGLAccounts, isLoading, isError } = useGLAccounts(showEditModal);

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Division Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {divisionCode && (
            <EditDCModalForm
              divisionCode={divisionCode}
              selectGlAccountData={selectGLAccounts}
              isGLAccountsError={isError}
              isGLAccountsLoading={isLoading}
            />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
