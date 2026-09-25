import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { SwitchField } from "@/components/fields/switch-field";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, getEndOfDay, getStartOfDay } from "@trenova/shared/lib/date";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import {
  accountingStartDateSchema,
  startDateInClosedBooks,
  type AccountingStartDateValues,
} from "./accounting-start-date-schema";
import type { AccountingVendor } from "./accounting-vendors";
import { useAccountingSyncSetupActions } from "./use-accounting-connection";

type AccountingStartDateStepProps = {
  vendor: AccountingVendor;
  connection: AccountingConnection;
  canManage: boolean;
};

export function AccountingStartDateStep({
  vendor,
  connection,
  canManage,
}: AccountingStartDateStepProps) {
  const t = useT();
  const latestAllowed = useMemo(() => getEndOfDay(), []);
  const form = useForm<AccountingStartDateValues>({
    resolver: zodResolver(accountingStartDateSchema(latestAllowed)),
    defaultValues: {
      startDate: connection.syncStartDate ?? getStartOfDay(),
      autoSync: connection.autoSync,
      driverSettlements: connection.syncsDriverSettlements,
      backfill: false,
    },
  });
  const { control, handleSubmit } = form;
  const { enable } = useAccountingSyncSetupActions(vendor, form);
  const startDate = useWatch({ control, name: "startDate" });
  const autoSync = useWatch({ control, name: "autoSync" });
  const closedThrough = connection.externalBooksClosedThrough;
  const beforeToday = startDate != null && startDate < getStartOfDay();

  return (
    <Form onSubmit={handleSubmit((values) => enable.mutate(values))} className="space-y-4">
      <div className="space-y-2">
        <h3 className="text-base font-semibold">{t("Choose when sending starts")}</h3>
        <p className="text-foreground-muted text-sm">
          {t(
            "Trenova sends invoices, credit and debit memos, customer payments, credit applications, carrier settlements and their payments to {0} as they are posted. Documents dated before the start date are never sent, so anything you already entered in {0} by hand is not duplicated.",
            vendor.name,
          )}
        </p>
      </div>
      <FormGroup cols={1}>
        <FormControl>
          <AutoCompleteDateField
            name="startDate"
            control={control}
            label={t("Start date")}
            description={t("The first document date Trenova sends.")}
            disabled={!canManage}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            name="autoSync"
            control={control}
            label={t("Send posted documents automatically")}
            description={
              autoSync
                ? t("Each document is sent as soon as it is posted.")
                : t("Each document waits in the sync ledger until someone releases it.")
            }
            disabled={!canManage}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            name="driverSettlements"
            control={control}
            label={t("Send owner-operator settlements")}
            description={t(
              "Sends owner-operator settlements to {0} as bills to a vendor for each driver. Company driver pay is never sent; it belongs to your payroll system.",
              vendor.name,
            )}
            disabled={!canManage}
          />
        </FormControl>
        {beforeToday ? (
          <FormControl>
            <CheckboxField
              name="backfill"
              control={control}
              label={t("Also send documents already posted since the start date")}
              description={t(
                "Queues the documents dated from the start date up to today. Leave this off if they are already in {0}.",
                vendor.name,
              )}
              disabled={!canManage}
            />
          </FormControl>
        ) : null}
      </FormGroup>
      {startDateInClosedBooks(startDate, closedThrough) ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t(
              "{0} has its books closed through {1}. Documents dated on or before that day will be held until the books are reopened or the documents are skipped.",
              vendor.name,
              formatUnixDateMedium(closedThrough),
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      {!canManage ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("Choosing the start date needs manage access to the accounting integration.")}
          </AlertDescription>
        </Alert>
      ) : null}
      <div className="flex justify-end gap-2 border-t pt-4">
        <Button
          type="submit"
          disabled={!canManage}
          isLoading={enable.isPending}
          loadingText={t("Starting...")}
        >
          {t("Start sending")}
        </Button>
      </div>
    </Form>
  );
}
