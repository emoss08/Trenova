import { SettingsForm } from "@/components/settings/settings-form";
import { accountingControl } from "@/lib/queries/accounting-control";
import { accountingControlService } from "@/services/accounting-control";
import { accountingControlFormSchema, type AccountingControl } from "@/types/accounting-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { sanitizeAccountingControlForSubmit } from "../accounting-control-submit";
import { useAccountingControlTabs, type AccountingControlTab } from "../accounting-control-tabs";
import { AccountsTab } from "./accounts-tab";
import { CloseTab } from "./close-tab";
import { CurrencyTab } from "./currency-tab";
import { DriverPayTab } from "./driver-pay-tab";
import { PostingTab } from "./posting-tab";

/**
 * Accounting control: one form, saved once, split into a tab for each job a person comes
 * here to do. Driver pay is there only for an organization that runs a fleet and pays its
 * drivers through Trenova.
 */
export default function AccountingControlForm() {
  const { data } = useSuspenseQuery({ ...accountingControl.get() });
  const tabs = useAccountingControlTabs();

  return (
    <SettingsForm<AccountingControl, AccountingControlTab>
      values={data}
      schema={accountingControlFormSchema}
      queryKey={accountingControl.get._def}
      save={(values) => accountingControlService.update(values)}
      resourceName="Accounting Control"
      prepare={sanitizeAccountingControlForSubmit}
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "posting":
            return <PostingTab />;
          case "accounts":
            return <AccountsTab />;
          case "driverPay":
            return <DriverPayTab />;
          case "close":
            return <CloseTab />;
          case "currency":
            return <CurrencyTab />;
        }
      }}
    />
  );
}
