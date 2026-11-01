import { ChoiceButton } from "@/components/choice-button";
import { ACCOUNTING_LEDGER_GRANULARITIES, ACCOUNTING_SYNC_MODES } from "@/lib/accounting-sync";
import type {
  AccountingConnection,
  AccountingProviderProfile,
} from "@/lib/graphql/accounting-sync";
import { zodResolver } from "@hookform/resolvers/zod";
import type {
  AccountingLedgerGranularity,
  AccountingSyncMode,
} from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { InfoCircleIcon, Lock01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { accountingModeSchema, type AccountingModeValues } from "./accounting-start-date-schema";
import type { AccountingVendor } from "./accounting-vendors";
import { useAccountingModeAction } from "./use-accounting-connection";
import { IntegrationSetupFooter } from "../shared/integration-setup-wizard";

type AccountingModeStepProps = {
  vendor: AccountingVendor;
  profile: AccountingProviderProfile;
  connection: AccountingConnection;
  canManage: boolean;
};

export function AccountingModeStep({
  vendor,
  profile,
  connection,
  canManage,
}: AccountingModeStepProps) {
  const t = useT();
  const ledgerAvailable = profile.ledgerAvailable;
  const schema = useMemo(() => accountingModeSchema(ledgerAvailable), [ledgerAvailable]);
  const form = useForm<AccountingModeValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      mode: ledgerAvailable ? connection.syncMode : "Document",
      granularity: connection.ledgerGranularity ?? "Detailed",
    },
  });
  const { control, handleSubmit, setValue } = form;
  const choose = useAccountingModeAction(vendor, ledgerAvailable, form);
  const mode = useWatch({ control, name: "mode" });
  const granularity = useWatch({ control, name: "granularity" });

  const modeChoices: Record<AccountingSyncMode, { label: string; description: string }> = {
    Document: {
      label: t("Send documents"),
      description: t(
        "Trenova sends {0} the invoices, memos, customer payments and settlement bills it posts, and {0} keeps the ledger.",
        vendor.name,
      ),
    },
    Ledger: {
      label: t("Send journal entries"),
      description: t(
        "Trenova keeps the ledger and sends {0} every journal entry it posts. Invoices, payments and bills stay in Trenova; {0} receives their accounting.",
        vendor.name,
      ),
    },
  };
  const granularityChoices: Record<
    AccountingLedgerGranularity,
    { label: string; description: string }
  > = {
    Detailed: {
      label: t("Detailed"),
      description: t("One journal entry in {0} for each one posted in Trenova.", vendor.name),
    },
    DailySummary: {
      label: t("Daily summary"),
      description: t(
        "One journal entry per day, summed per account and customer or vendor. A day is sent again when an entry is posted to it later.",
      ),
    },
  };

  return (
    <Form
      onSubmit={handleSubmit((values) => choose.mutate(values))}
      className="flex flex-1 flex-col gap-4"
    >
      <h3 className="text-base font-semibold">{t("Choose what is sent")}</h3>
      <div role="radiogroup" aria-label={t("What is sent")} className="grid gap-2 sm:grid-cols-2">
        {ACCOUNTING_SYNC_MODES.map((value) => {
          const unavailable = value === "Ledger" && !ledgerAvailable;
          return (
            <ChoiceButton
              key={value}
              selected={mode === value}
              disabled={!canManage || unavailable}
              onClick={() => setValue("mode", value, { shouldDirty: true })}
              className="gap-1 p-3"
            >
              <span className="font-medium">{modeChoices[value].label}</span>
              <span className="text-foreground-muted text-xs">
                {unavailable
                  ? t("Not available for {0}", vendor.name)
                  : modeChoices[value].description}
              </span>
            </ChoiceButton>
          );
        })}
      </div>
      {mode === "Ledger" && ledgerAvailable ? (
        <div className="space-y-2">
          <p className="text-sm font-medium">{t("Journal entries")}</p>
          <div
            role="radiogroup"
            aria-label={t("Journal entries")}
            className="grid gap-2 sm:grid-cols-2"
          >
            {ACCOUNTING_LEDGER_GRANULARITIES.map((value) => (
              <ChoiceButton
                key={value}
                selected={granularity === value}
                disabled={!canManage}
                onClick={() => setValue("granularity", value, { shouldDirty: true })}
                className="gap-1 p-3"
              >
                <span className="font-medium">{granularityChoices[value].label}</span>
                <span className="text-foreground-muted text-xs">
                  {granularityChoices[value].description}
                </span>
              </ChoiceButton>
            ))}
          </div>
        </div>
      ) : null}
      {!ledgerAvailable ? (
        <Alert size="sm" variant="info">
          <AlertTitle>{t("Journal entries cannot be sent to {0}", vendor.name)}</AlertTitle>
          <AlertDescription>
            {profile.ledgerUnavailableReason
              ? t(profile.ledgerUnavailableReason)
              : t("{0} can only receive documents.", vendor.name)}
          </AlertDescription>
        </Alert>
      ) : null}
      <ul className="bg-muted/40 text-foreground-muted space-y-2 rounded-lg border p-3 text-sm">
        {mode === "Ledger" && ledgerAvailable ? (
          <li className="flex gap-2">
            <InfoCircleIcon
              aria-hidden
              className="text-foreground-subtle mt-0.5 size-3.5 shrink-0"
            />
            <span>
              {t(
                "Entries are sent once they are posted. When journals are posted by hand, an entry goes to {0} after someone posts it on Journals to post.",
                vendor.name,
              )}
            </span>
          </li>
        ) : null}
        <li className="flex gap-2">
          <Lock01Icon aria-hidden className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />
          <span>
            {t(
              "This choice is fixed once sending starts, because {0} will already hold what was sent.",
              vendor.name,
            )}
          </span>
        </li>
      </ul>
      {!canManage ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("Choosing what is sent needs manage access to the accounting integration.")}
          </AlertDescription>
        </Alert>
      ) : null}
      <IntegrationSetupFooter>
        <Button
          type="submit"
          disabled={!canManage}
          isLoading={choose.isPending}
          loadingText={t("Saving...")}
        >
          {t("Continue")}
        </Button>
      </IntegrationSetupFooter>
    </Form>
  );
}
