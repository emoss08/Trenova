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

import { useFormStyles } from "@/assets/styles/FormStyles";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useGLAccounts } from "@/hooks/useGLAccounts";
import { useTags } from "@/hooks/useTags";
import { useUsers } from "@/hooks/useUsers";
import { glAccountSchema } from "@/lib/validations/AccountingSchema";
import { generalLedgerTableStore as store } from "@/stores/AccountingStores";
import {
  GLAccountFormValues as FormValues,
  GeneralLedgerAccount,
} from "@/types/accounting";
import { TableStoreProps } from "@/types/tables";
import { Button, Drawer, Group } from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import React from "react";
import { GLAccountForm } from "./CreateGLAccountModal";

export function EditGLAccountModalForm({
  glAccount,
  onCancel,
}: {
  glAccount: GeneralLedgerAccount;
  onCancel: () => void;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const [drawerOpen] = store.use("drawerOpen");

  const form = useForm<FormValues>({
    validate: yupResolver(glAccountSchema),
    initialValues: {
      status: glAccount.status,
      accountNumber: glAccount.accountNumber,
      description: glAccount.description,
      accountType: glAccount.accountType,
      cashFlowType: glAccount.cashFlowType,
      accountSubType: glAccount.accountSubType,
      accountClassification: glAccount.accountClassification,
      balance: glAccount.balance,
      openingBalance: glAccount.openingBalance,
      closingBalance: glAccount.closingBalance,
      parentAccount: glAccount.parentAccount,
      isReconciled: glAccount.isReconciled,
      notes: glAccount.notes,
      isTaxRelevant: glAccount.isTaxRelevant,
      attachment: glAccount.attachment,
      interestRate: glAccount.interestRate,
      tags: glAccount.tags,
    },
  });

  console.info("Form Values", form.values);

  const mutation = useCustomMutation<
    FormValues,
    TableStoreProps<GeneralLedgerAccount>
  >(
    form,
    notifications,
    {
      method: "PUT",
      path: `/gl_accounts/${glAccount.id}/`,
      successMessage: "General Ledger Account updated successfully.",
      queryKeysToInvalidate: ["gl-account-table-data"],
      additionalInvalidateQueries: ["glAccounts"],
      closeModal: true,
      errorMessage: "Failed to create general ledger account.",
    },
    () => setLoading(false),
    store,
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    const dataToSend = { ...values };

    if (
      typeof dataToSend.attachment === "string" ||
      dataToSend.attachment === null
    ) {
      delete dataToSend.attachment; // Don't send if it's the existing URL or null
    }

    mutation.mutate(dataToSend);
  };

  const {
    selectUsersData,
    isError: usersError,
    isLoading: usersLoading,
  } = useUsers(drawerOpen);

  const {
    selectTags,
    isError: tagsError,
    isLoading: tagsLoading,
  } = useTags(drawerOpen);

  const {
    selectGLAccounts,
    isError: glAccountsError,
    isLoading: glAccountsLoading,
  } = useGLAccounts(drawerOpen);

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <GLAccountForm
        users={selectUsersData}
        isUsersError={usersError}
        isUsersLoading={usersLoading}
        tags={selectTags}
        isTagsError={tagsError}
        isTagsLoading={tagsLoading}
        glAccounts={selectGLAccounts}
        isGLAccountsLoading={glAccountsError}
        isGLAccountsError={glAccountsLoading}
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

export function GLAccountDrawer(): React.ReactElement {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [glAccount] = store.use("selectedRecord");

  const onCancel = () => setDrawerOpen(false);

  return (
    <Drawer.Root
      position="right"
      opened={drawerOpen}
      onClose={() => setDrawerOpen(false)}
      size="50%"
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>
            Edit GL Account: {glAccount && glAccount.accountNumber}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {glAccount && (
            <EditGLAccountModalForm glAccount={glAccount} onCancel={onCancel} />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
