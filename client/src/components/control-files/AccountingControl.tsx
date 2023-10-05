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
  Card,
  Divider,
  Group,
  SimpleGrid,
  Skeleton,
  Text,
} from "@mantine/core";
import React from "react";
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { usePageStyles } from "@/assets/styles/PageStyles";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { SwitchInput } from "@/components/common/fields/SwitchInput";
import { SelectInput } from "@/components/common/fields/SelectInput";
import {
  AccountingControl,
  AccountingControlFormValues as FormValues,
} from "@/types/accounting";
import { accountingControlSchema } from "@/lib/schemas/AccountingSchema";
import { useAccountingControl } from "@/hooks/useAccounting";
import {
  automaticJournalEntryChoices,
  thresholdActionChoices,
} from "@/lib/choices";
import { TChoiceProps } from "@/types";
import { useGLAccounts } from "@/hooks/useGLAccounts";
import { ValidatedMultiSelect } from "@/components/common/fields/MultiSelect";
import { useUsers } from "@/hooks/useUsers";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { useCustomMutation } from "@/hooks/useCustomMutation";

function AccountingControlForm({
  accountingControl,
  glAccounts,
  isGlAccountsLoading,
  isGlAccountsError,
  users,
  isUsersLoading,
  isUsersError,
}: {
  accountingControl: AccountingControl;
  glAccounts: TChoiceProps[];
  isGlAccountsLoading: boolean;
  isGlAccountsError: boolean;
  users: TChoiceProps[];
  isUsersLoading: boolean;
  isUsersError: boolean;
}): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(accountingControlSchema),
    initialValues: {
      autoCreateJournalEntries: accountingControl.autoCreateJournalEntries,
      journalEntryCriteria: accountingControl.journalEntryCriteria,
      restrictManualJournalEntries:
        accountingControl.restrictManualJournalEntries,
      requireJournalEntryApproval:
        accountingControl.requireJournalEntryApproval,
      defaultExpenseAccount: accountingControl.defaultExpenseAccount,
      defaultRevenueAccount: accountingControl.defaultRevenueAccount,
      enableReconciliationNotifications:
        accountingControl.enableReconciliationNotifications,
      reconciliationNotificationRecipients:
        accountingControl.reconciliationNotificationRecipients,
      reconciliationThreshold: accountingControl.reconciliationThreshold,
      reconciliationThresholdAction:
        accountingControl.reconciliationThresholdAction,
      haltOnPendingReconciliation:
        accountingControl.haltOnPendingReconciliation,
      criticalProcesses: accountingControl.criticalProcesses,
    },
  });

  const mutation = useCustomMutation<FormValues, undefined>(
    form,
    notifications,
    {
      method: "PUT",
      path: `/accounting_control/${accountingControl.id}/`,
      successMessage: "Accounting control updated successfully.",
      queryKeysToInvalidate: ["accessorialCharges"],
      closeModal: false,
      errorMessage: "Failed to update accounting control.",
    },
    () => setLoading(false),
  );

  const handleSubmit = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SwitchInput<FormValues>
              form={form}
              className={classes.fields}
              name="autoCreateJournalEntries"
              label="Auto Create Journal Entries"
              description="Whether to automatically create journal entries based on certain triggers"
            />
            <SelectInput<FormValues>
              form={form}
              data={automaticJournalEntryChoices}
              name="journalEntryCriteria"
              label="Automatically Criteria"
              placeholder="Automatically Criteria"
              description="Define a criteria on when automatic journal entries are to be created"
            />
            <SwitchInput<FormValues>
              form={form}
              className={classes.fields}
              name="restrictManualJournalEntries"
              label="Restrict Manual Journal Entries"
              description="If set to True, users will not be able to manually create journal entries without specific permissions"
            />
            <SwitchInput<FormValues>
              form={form}
              className={classes.fields}
              name="requireJournalEntryApproval"
              label="Require Approval for Journal Entries"
              description="If set to True, all created journal entries will need to be reviewed and approved by authorized personnel before being finalized"
            />
            <SelectInput<FormValues>
              form={form}
              data={glAccounts}
              isLoading={isGlAccountsLoading}
              isError={isGlAccountsError}
              name="defaultRevenueAccount"
              description="Default revenue account if no specific RevenueCode is provided"
              label="Default Revenue Account"
              placeholder="Default Revenue Account"
            />
            <SelectInput<FormValues>
              form={form}
              data={glAccounts}
              isLoading={isGlAccountsLoading}
              isError={isGlAccountsError}
              name="defaultExpenseAccount"
              label="Default Expense Account"
              description="Default expense account if no specific RevenueCode is provided"
              placeholder="Default Expense Account"
            />
            <SwitchInput<FormValues>
              form={form}
              className={classes.fields}
              name="enableReconciliationNotifications"
              label="Enable Reconciliation Notifications"
              description="Send notifications when shipments are added to the reconciliation queue"
            />
            <ValidatedMultiSelect<FormValues>
              form={form}
              data={users}
              isError={isUsersError}
              isLoading={isUsersLoading}
              name="reconciliationNotificationRecipients"
              label="Reconciliation Notification Recipients"
              placeholder="Reconciliation Notification Recipients"
              description="Users who will receive notifications about reconciliation tasks. Leave empty for default recipients"
            />
            <ValidatedNumberInput<FormValues>
              form={form}
              name="reconciliationThreshold"
              placeholder="Reconciliation Threshold"
              label="Reconciliation Threshold"
              description="Threshold for pending reconciliation tasks. If exceeded, can trigger warnings or halt certain processes"
              withAsterisk
            />
            <SelectInput<FormValues>
              form={form}
              data={thresholdActionChoices}
              placeholder="Reconciliation Threshold Action"
              name="reconciliationThresholdAction"
              label="Reconciliation Threshold Action"
              description="Action to take when the reconciliation threshold is exceeded. Can be used to halt certain processes"
              withAsterisk
            />
            <SwitchInput<FormValues>
              form={form}
              className={classes.fields}
              name="haltOnPendingReconciliation"
              label="Halt on Pending Reconciliation"
              description="Halt critical processes if there are pending reconciliation tasks above the threshold"
            />
          </SimpleGrid>
          <ValidatedTextArea<FormValues>
            form={form}
            mt={10}
            label="Critical Processes"
            placeholder="Critical Processes"
            description="List of critical processes that shouldn't proceed if pending reconciliation tasks are above the threshold. Define clear identifiers or names for each process"
            name="criticalProcesses"
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

export default function AccountingControlPage(): React.ReactElement {
  const { classes } = usePageStyles();
  const { dataArray, isLoading } = useAccountingControl();
  const {
    selectGLAccounts,
    isLoading: isGLAccountsLoading,
    isError: isGLAccountsError,
  } = useGLAccounts();
  const {
    selectUsersData,
    isError: isUsersError,
    isLoading: isUsersLoading,
  } = useUsers();

  return isLoading ? (
    <Skeleton height={400} />
  ) : (
    <Card className={classes.card}>
      <Text fz="xl" fw={700} className={classes.text}>
        Accounting Controls
      </Text>

      <Divider my={10} />
      {dataArray && (
        <AccountingControlForm
          accountingControl={dataArray}
          glAccounts={selectGLAccounts}
          isGlAccountsLoading={isGLAccountsLoading}
          isGlAccountsError={isGLAccountsError}
          users={selectUsersData}
          isUsersLoading={isUsersLoading}
          isUsersError={isUsersError}
        />
      )}
    </Card>
  );
}
