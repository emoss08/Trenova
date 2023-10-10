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

import { Button, Drawer, Group } from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { divisionCodeTableStore as store } from "@/stores/AccountingStores";
import {
  DivisionCode,
  DivisionCodeFormValues as FormValues,
} from "@/types/accounting";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { divisionCodeSchema } from "@/lib/validations/AccountingSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { useGLAccounts } from "@/hooks/useGLAccounts";
import { DCForm } from "@/components/division-codes/table/CreateDCModal";

type EditDCModalFormProps = {
  divisionCode: DivisionCode;
  selectGlAccountData: TChoiceProps[];
  isGLAccountsLoading: boolean;
  isGLAccountsError: boolean;
  onCancel: () => void;
};

export function EditDCModalForm({
  divisionCode,
  selectGlAccountData,
  isGLAccountsError,
  isGLAccountsLoading,
  onCancel,
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

  const mutation = useCustomMutation<FormValues, TableStoreProps<DivisionCode>>(
    form,
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
        isError={isGLAccountsError}
        isLoading={isGLAccountsLoading}
        form={form}
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

export function DCDrawer(): React.ReactElement {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [divisionCode] = store.use("selectedRecord");
  const { selectGLAccounts, isError, isLoading } = useGLAccounts(drawerOpen);

  const onCancel = () => setDrawerOpen(false);

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
            Edit Division Code: {divisionCode && divisionCode.code}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {divisionCode && (
            <EditDCModalForm
              divisionCode={divisionCode}
              selectGlAccountData={selectGLAccounts}
              isGLAccountsError={isError}
              isGLAccountsLoading={isLoading}
              onCancel={onCancel}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}