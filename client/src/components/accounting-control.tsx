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
import type {
  AccountingControlFormValues,
  AccountingControl as AccountingControlType,
} from "@/types/accounting";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { ErrorLoadingData } from "./common/table/data-table-components";

function AccountingControlForm({
  accountingControl,
}: {
  accountingControl: AccountingControlType;
}) {
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
      defaultValues: accountingControl,
    },
  );

  const mutation = useCustomMutation<AccountingControlFormValues>(control, {
    method: "PUT",
    path: `/accounting-control/${accountingControl.id}/`,
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "accountingControl",
    reset,
    errorMessage: t("formErrorMessage"),
  });

  const onSubmit = (values: AccountingControlFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
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
              name="defaultRevAccountId"
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
              name="defaultExpAccountId"
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
              name="enableRecNotifications"
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
              name="recThreshold"
              control={control}
              label={t("fields.reconciliationThreshold.label")}
              placeholder={t("fields.reconciliationThreshold.placeholder")}
              description={t("fields.reconciliationThreshold.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="recThresholdAction"
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
              name="haltOnPendingRec"
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
      <div className="border-border flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={mutation.isPending}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={mutation.isPending}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function AccountingControl() {
  const { data, isLoading, isError } = useAccountingControl();
  const { t } = useTranslation(["admin.accountingcontrol"]);

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-foreground text-base font-semibold leading-7">
          {t("title")}
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          {t("subTitle")}
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="bg-background ring-muted m-4 p-8 ring-1 sm:rounded-xl md:col-span-2">
          <ErrorLoadingData />
        </div>
      ) : (
        data && <AccountingControlForm accountingControl={data} />
      )}
    </div>
  );
}
