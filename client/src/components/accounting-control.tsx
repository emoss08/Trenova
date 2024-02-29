/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  useAccountingControl,
  useGLAccounts,
  useUsers,
} from "@/hooks/useQueries";
import {
  automaticJournalEntryChoices,
  thresholdActionChoices,
} from "@/lib/choices";
import { accountingControlSchema } from "@/lib/validations/AccountingSchema";
import {
  AccountingControl as AccountingControlType,
  AccountingControlFormValues,
} from "@/types/accounting";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function AccountingControlForm({
  accountingControl,
}: {
  accountingControl: AccountingControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { t } = useTranslation(["admin.accountingcontrol", "common"]);

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
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["accountingControl"],
      errorMessage: t("formErrorMessage"),
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
      className="m-4 border border-border bg-card sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-full">
            <SelectInput
              name="journalEntryCriteria"
              control={control}
              options={automaticJournalEntryChoices}
              rules={{ required: true }}
              label={t("fields.journalEntryCriteria.label")}
              placeholder={t("fields.journalEntryCriteria.placeholder")}
              description={t("fields.journalEntryCriteria.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="defaultRevenueAccount"
              control={control}
              options={selectGLAccounts}
              isLoading={isGLAccountsLoading}
              isFetchError={isGlAccountsError}
              label={t("fields.defaultRevenueAccount.label")}
              placeholder={t("fields.defaultRevenueAccount.placeholder")}
              description={t("fields.defaultRevenueAccount.description")}
              hasPopoutWindow
              popoutLink="/accounting/gl-accounts"
              isClearable
              popoutLinkLabel="GL Account"
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="defaultExpenseAccount"
              control={control}
              options={selectGLAccounts}
              isLoading={isGLAccountsLoading}
              isFetchError={isGlAccountsError}
              label={t("fields.defaultExpenseAccount.label")}
              placeholder={t("fields.defaultExpenseAccount.placeholder")}
              description={t("fields.defaultExpenseAccount.description")}
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
              label={t("fields.autoCreateJournalEntries.label")}
              description={t("fields.autoCreateJournalEntries.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="restrictManualJournalEntries"
              control={control}
              label={t("fields.restrictManualJournalEntries.label")}
              description={t("fields.restrictManualJournalEntries.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="requireJournalEntryApproval"
              control={control}
              label={t("fields.requireJournalEntryApproval.label")}
              description={t("fields.requireJournalEntryApproval.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enableReconciliationNotifications"
              control={control}
              label={t("fields.enableReconciliationNotifications.label")}
              description={t(
                "fields.enableReconciliationNotifications.description",
              )}
            />
          </div>
          <div className="col-span-full">
            <SelectInput
              name="reconciliationNotificationRecipients"
              control={control}
              options={selectUsersData}
              isLoading={isUsersLoading}
              isFetchError={isUsersError}
              label={t("fields.reconciliationNotificationRecipients.label")}
              placeholder={t(
                "fields.reconciliationNotificationRecipients.placeholder",
              )}
              description={t(
                "fields.reconciliationNotificationRecipients.description",
              )}
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
              label={t("fields.reconciliationThreshold.label")}
              placeholder={t("fields.reconciliationThreshold.placeholder")}
              description={t("fields.reconciliationThreshold.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="reconciliationThresholdAction"
              control={control}
              options={thresholdActionChoices}
              rules={{ required: true }}
              label={t("fields.reconciliationThresholdAction.label")}
              placeholder={t(
                "fields.reconciliationThresholdAction.placeholder",
              )}
              description={t(
                "fields.reconciliationThresholdAction.description",
              )}
            />
          </div>
          <div className="col-span-full">
            <CheckboxInput
              name="haltOnPendingReconciliation"
              control={control}
              label={t("fields.haltOnPendingReconciliation.label")}
              description={t("fields.haltOnPendingReconciliation.description")}
            />
          </div>
          <div className="col-span-full">
            <TextareaField
              name="criticalProcesses"
              control={control}
              label={t("fields.criticalProcesses.label")}
              placeholder={t("fields.criticalProcesses.placeholder")}
              description={t("fields.criticalProcesses.description")}
              readOnly
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-4 border-t border-border p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={isSubmitting}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function AccountingControl() {
  const { accountingControlData, isLoading } = useAccountingControl();
  const { t } = useTranslation(["admin.accountingcontrol"]);

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          {t("title")}
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          {t("subTitle")}
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
