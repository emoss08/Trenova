/*
 * COPYRIGHT(c) 2024 MONTA
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

import { Button } from "@/components/ui/button";
import {
  useAccountingControl,
  useGLAccounts,
  useUsers,
} from "@/hooks/useQueries";
import React from "react";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  AccountingControl as AccountingControlType,
  AccountingControlFormValues,
} from "@/types/accounting";
import { SelectInput } from "@/components/common/fields/select-input";
import {
  automaticJournalEntryChoices,
  thresholdActionChoices,
} from "@/lib/choices";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { yupResolver } from "@hookform/resolvers/yup";
import { accountingControlSchema } from "@/lib/validations/accounting";
import { AsyncSelectInput } from "@/components/common/fields/async-select-input";
import { Skeleton } from "@/components/ui/skeleton";

function AccountingControlForm({
  accountingControl,
}: {
  accountingControl: AccountingControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const {
    selectGLAccounts,
    isLoading: isGLAccountsLoading,
    isError: isGlAccountsError,
  } = useGLAccounts();
  const {
    selectUsersData,
    isError: isUsersError,
    isLoading: isUsersLoading,
  } = useUsers();

  const { control, handleSubmit, reset } = useForm<AccountingControlFormValues>(
    {
      resolver: yupResolver(accountingControlSchema),
      defaultValues: {
        autoCreateJournalEntries: accountingControl.autoCreateJournalEntries,
        journalEntryCriteria: accountingControl.journalEntryCriteria,
        restrictManualJournalEntries:
          accountingControl.restrictManualJournalEntries,
        requireJournalEntryApproval:
          accountingControl.requireJournalEntryApproval,
        defaultRevenueAccount: accountingControl.defaultRevenueAccount,
        defaultExpenseAccount: accountingControl.defaultExpenseAccount,
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
    },
  );

  const mutation = useCustomMutation<AccountingControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/accounting_control/${accountingControl.id}/`,
      successMessage: "Accounting Control updated successfully.",
      queryKeysToInvalidate: ["accountingControl"],
      errorMessage: "Failed to update accounting control.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: AccountingControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <form
      className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
          <div className="col-span-full">
            <SelectInput
              name="journalEntryCriteria"
              control={control}
              options={automaticJournalEntryChoices}
              rules={{ required: true }}
              label="Journal Entry Criteria"
              placeholder="Journal Entry Criteria"
              description="Set automated rules for journal entry creation to enhance financial recording efficiency."
            />
          </div>
          <div className="col-span-3">
            <AsyncSelectInput
              name="defaultRevenueAccount"
              control={control}
              options={selectGLAccounts}
              isLoading={isGLAccountsLoading}
              isFetchError={isGlAccountsError}
              label="Default Revenue Account"
              placeholder="Default Revenue Account"
              description="Select a default revenue account for shipments lacking a specific RevenueCode."
              hasPopoutWindow
              popoutLink="/accounting/gl-accounts"
              isClearable
              popoutLinkLabel="GL Account"
            />
          </div>
          <div className="col-span-3">
            <AsyncSelectInput
              name="defaultExpenseAccount"
              control={control}
              options={selectGLAccounts}
              isLoading={isGLAccountsLoading}
              isFetchError={isGlAccountsError}
              label="Default Expense Account"
              placeholder="Default Expense Account"
              description="Choose a fallback expense account for shipments without a designated RevenueCode."
              hasPopoutWindow
              popoutLink="/accounting/gl-accounts"
              isClearable
              popoutLinkLabel="GL Account"
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoCreateJournalEntries"
              control={control}
              label="Auto Create Journal Entries"
              description="Enable the system to automatically generate journal entries based on predefined triggers."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="restrictManualJournalEntries"
              control={control}
              label="Restrict Manual Journal Entries"
              description="Toggle to restrict manual journal entries creation to authorized personnel only."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="requireJournalEntryApproval"
              control={control}
              label="Require Journal Entry Approval"
              description="Activate mandatory approval for all journal entries by designated authorities before posting."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enableReconciliationNotifications"
              control={control}
              label="Enable Reconciliation Notifications"
              description="Enable alerts for the addition of shipments to the reconciliation queue."
            />
          </div>
          <div className="col-span-full">
            <SelectInput
              name="reconciliationNotificationRecipients"
              control={control}
              options={selectUsersData}
              isLoading={isUsersLoading}
              isFetchError={isUsersError}
              label="Reconciliation Notification Recipients"
              placeholder="Reconciliation Notification Recipients"
              description="Designate users to receive alerts regarding new shipments in the reconciliation queue."
              isMulti
              hasPopoutWindow
              popoutLink="#"
              isClearable
              popoutLinkLabel="User"
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="reconciliationThreshold"
              control={control}
              label="Reconciliation Threshold"
              placeholder="Reconciliation Threshold"
              description="Set a threshold for pending reconciliation tasks, triggering alerts or process halts when exceeded."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="reconciliationThresholdAction"
              control={control}
              options={thresholdActionChoices}
              rules={{ required: true }}
              label="Reconciliation Threshold Action"
              placeholder="Reconciliation Threshold Action"
              description="Define the actions to be taken when the set reconciliation threshold is reached."
            />
          </div>
          <div className="col-span-full">
            <CheckboxInput
              name="haltOnPendingReconciliation"
              control={control}
              label="Halt on Pending Reconciliation"
              description="Stop essential operations if pending reconciliation tasks surpass the defined threshold."
            />
          </div>
          <div className="col-span-full">
            <TextareaField
              name="criticalProcesses"
              control={control}
              label="Critical Processes"
              description="Enumerate crucial operations that are to be paused when exceeding the reconciliation task threshold."
              disabled
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-6 border-t border-gray-900/10 p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            console.log("cancel");
          }}
          type="button"
          variant="ghost"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function AccountingControl() {
  const { accountingControlData, isLoading } = useAccountingControl();

  return (
    <div className="grid grid-cols-1 gap-8 pt-10 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Accounting Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Streamline your financial operations with precision and compliance.
          This module centralizes key financial elements of your transportation
          business, ensuring accurate tracking, seamless communication, and
          tailored service delivery within the transportation industry.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : (
        accountingControlData && (
          <AccountingControlForm accountingControl={accountingControlData} />
        )
      )}
    </div>
  );
}
