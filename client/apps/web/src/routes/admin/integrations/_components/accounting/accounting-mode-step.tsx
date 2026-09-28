import { SelectField } from "@/components/fields/select-field";
import { ACCOUNTING_LEDGER_GRANULARITIES, ACCOUNTING_SYNC_MODES } from "@/lib/accounting-sync";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { zodResolver } from "@hookform/resolvers/zod";
import type {
  AccountingLedgerGranularity,
  AccountingSyncMode,
} from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { accountingModeSchema, type AccountingModeValues } from "./accounting-start-date-schema";
import type { AccountingVendor } from "./accounting-vendors";
import { useAccountingModeAction } from "./use-accounting-connection";

type AccountingModeStepProps = {
  vendor: AccountingVendor;
  connection: AccountingConnection;
  canManage: boolean;
};

export function AccountingModeStep({ vendor, connection, canManage }: AccountingModeStepProps) {
  const t = useT();
  const form = useForm<AccountingModeValues>({
    resolver: zodResolver(accountingModeSchema),
    defaultValues: {
      mode: connection.syncMode,
      granularity: connection.ledgerGranularity ?? "Detailed",
    },
  });
  const { control, handleSubmit } = form;
  const choose = useAccountingModeAction(vendor, form);
  const mode = useWatch({ control, name: "mode" });
  const granularity = useWatch({ control, name: "granularity" });

  const modeLabels = useMemo<Record<AccountingSyncMode, string>>(
    () => ({
      Document: t("Send documents"),
      Ledger: t("Send journal entries"),
    }),
    [t],
  );
  const granularityLabels = useMemo<Record<AccountingLedgerGranularity, string>>(
    () => ({
      Detailed: t("Detailed"),
      DailySummary: t("Daily summary"),
    }),
    [t],
  );

  return (
    <Form onSubmit={handleSubmit((values) => choose.mutate(values))} className="space-y-4">
      <div className="space-y-2">
        <h3 className="text-base font-semibold">{t("Choose what is sent")}</h3>
        <p className="text-foreground-muted text-sm">
          {mode === "Ledger"
            ? t(
                "Trenova keeps the ledger and sends {0} every journal entry it posts. Invoices, payments and bills stay in Trenova; {0} receives their accounting.",
                vendor.name,
              )
            : t(
                "Trenova sends {0} the invoices, memos, customer payments and settlement bills it posts, and {0} keeps the ledger.",
                vendor.name,
              )}
        </p>
      </div>
      <FormGroup cols={1}>
        <FormControl>
          <SelectField
            name="mode"
            control={control}
            label={t("What is sent")}
            options={ACCOUNTING_SYNC_MODES.map((value) => ({
              value,
              label: modeLabels[value],
            }))}
            isReadOnly={!canManage}
          />
        </FormControl>
        {mode === "Ledger" ? (
          <FormControl>
            <SelectField
              name="granularity"
              control={control}
              label={t("Journal entries")}
              description={
                granularity === "DailySummary"
                  ? t(
                      "One journal entry per day, summed per account and customer or vendor. A day is sent again when an entry is posted to it later.",
                    )
                  : t("One journal entry in {0} for each one posted in Trenova.", vendor.name)
              }
              options={ACCOUNTING_LEDGER_GRANULARITIES.map((value) => ({
                value,
                label: granularityLabels[value],
              }))}
              isReadOnly={!canManage}
            />
          </FormControl>
        ) : null}
      </FormGroup>
      {mode === "Ledger" ? (
        <Alert size="sm">
          <AlertDescription>
            {t(
              "Entries are sent once they are posted. When journals are posted by hand, an entry goes to {0} after someone posts it on Journals to post.",
              vendor.name,
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      <Alert size="sm" variant="warning">
        <AlertDescription>
          {t(
            "This choice is fixed once sending starts. Changing it later means disconnecting {0} and connecting again.",
            vendor.name,
          )}
        </AlertDescription>
      </Alert>
      {!canManage ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("Choosing what is sent needs manage access to the accounting integration.")}
          </AlertDescription>
        </Alert>
      ) : null}
      <div className="flex justify-end gap-2 border-t pt-4">
        <Button
          type="submit"
          disabled={!canManage}
          isLoading={choose.isPending}
          loadingText={t("Saving...")}
        >
          {t("Continue")}
        </Button>
      </div>
    </Form>
  );
}
