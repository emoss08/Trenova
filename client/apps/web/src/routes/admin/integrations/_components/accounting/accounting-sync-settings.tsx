import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useAccountingInboundLabels } from "@/hooks/use-accounting-inbound-labels";
import { ACCOUNTING_INBOUND_PATH, ACCOUNTING_INBOUND_POLICIES } from "@/lib/accounting-sync";
import type { AccountingInboundPaymentPolicy } from "@trenova/graphql/generated/graphql";
import { SectionPanel } from "@/components/section-panel";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { SlidersHorizontalIcon } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";
import { Link } from "react-router";
import {
  accountingSyncSettingsSchema,
  type AccountingSyncSettingsValues,
} from "./accounting-start-date-schema";
import type { AccountingVendor } from "./accounting-vendors";
import { useAccountingSyncSettingsAction } from "./use-accounting-connection";

type AccountingSyncSettingsProps = {
  vendor: AccountingVendor;
  connection: AccountingConnection;
  canManage: boolean;
};

export function AccountingSyncSettings({
  vendor,
  connection,
  canManage,
}: AccountingSyncSettingsProps) {
  const t = useT();
  const form = useForm<AccountingSyncSettingsValues>({
    resolver: zodResolver(accountingSyncSettingsSchema),
    values: {
      autoSync: connection.autoSync,
      driverSettlements: connection.syncsDriverSettlements,
      inboundPayments: connection.inboundPaymentPolicy,
    },
  });
  const inboundLabels = useAccountingInboundLabels();
  const {
    control,
    handleSubmit,
    reset,
    formState: { isDirty },
  } = form;
  const save = useAccountingSyncSettingsAction(vendor, form);
  const autoSync = useWatch({ control, name: "autoSync" });
  const driverSettlements = useWatch({ control, name: "driverSettlements" });
  const turningDriversOn = driverSettlements && !connection.syncsDriverSettlements;
  const inboundPayments = useWatch({ control, name: "inboundPayments" });
  const policyHelp: Record<AccountingInboundPaymentPolicy, string> = {
    Propose: t(
      "Each payment waits on the payments from the books page until someone applies or ignores it.",
    ),
    Apply: t(
      "Payments that match open invoices and settlements are applied without anyone looking at them. The rest still wait for a person.",
    ),
    Off: t(
      "Payments recorded in {0} are not read. Record them in Trenova by hand so invoices and settlements show as paid.",
      vendor.name,
    ),
  };

  return (
    <SectionPanel title={t("Sync settings")} icon={<SlidersHorizontalIcon />}>
      <Form onSubmit={handleSubmit((values) => save.mutate(values))} className="space-y-3 p-3">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              name="autoSync"
              control={control}
              label={t("Send posted documents automatically")}
              description={
                autoSync
                  ? t("Each document is sent as soon as it is posted.")
                  : t(
                      "Each document waits in the sync ledger until someone releases it. Documents already held stay held.",
                    )
              }
              disabled={!canManage}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              name="driverSettlements"
              control={control}
              label={t("Send owner-operator settlements")}
              description={
                connection.driverSettlementsEnabledAt && driverSettlements
                  ? t(
                      "Sent as bills since {0}. Company driver pay is never sent.",
                      formatUnixDateMedium(connection.driverSettlementsEnabledAt),
                    )
                  : t(
                      "Sends owner-operator settlements to {0} as bills to a vendor for each driver. Company driver pay is never sent; it belongs to your payroll system.",
                      vendor.name,
                    )
              }
              disabled={!canManage}
            />
          </FormControl>
          <FormControl>
            <SelectField
              name="inboundPayments"
              control={control}
              label={t("Payments recorded in {0}", vendor.name)}
              description={policyHelp[inboundPayments]}
              options={ACCOUNTING_INBOUND_POLICIES.map((value) => ({
                value,
                label: inboundLabels.policy[value],
              }))}
              isReadOnly={!canManage}
            />
          </FormControl>
        </FormGroup>
        {connection.inboundPaymentPolicy !== "Off" ? (
          <p className="text-foreground-muted text-xs">
            {t("Review them on")}{" "}
            <Link to={ACCOUNTING_INBOUND_PATH} className="text-brand font-medium hover:underline">
              {t("Payments from the books")}
            </Link>
          </p>
        ) : null}
        {turningDriversOn ? (
          <Alert size="sm" variant="info">
            <AlertDescription>
              {t(
                "Settlements posted from now on are sent. To send ones posted earlier, request a backfill from the sync ledger.",
              )}
            </AlertDescription>
          </Alert>
        ) : null}
        {!canManage ? (
          <Alert size="sm" variant="warning">
            <AlertDescription>
              {t(
                "Changing how documents are sent needs manage access to the accounting integration.",
              )}
            </AlertDescription>
          </Alert>
        ) : null}
        {canManage ? (
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={!isDirty || save.isPending}
              onClick={() => reset()}
            >
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              disabled={!isDirty}
              isLoading={save.isPending}
              loadingText={t("Saving...")}
            >
              {t("Save")}
            </Button>
          </div>
        ) : null}
      </Form>
    </SectionPanel>
  );
}
