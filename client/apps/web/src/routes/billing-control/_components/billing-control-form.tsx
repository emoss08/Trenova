import { SettingsForm } from "@/components/settings/settings-form";
import { billingControl } from "@/lib/queries/billing-control";
import { billingControlService } from "@/services/billing-control";
import { billingControlSchema, type BillingControl } from "@/types/billing-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { prepareBillingControlForSubmit } from "../billing-control-submit";
import { useBillingControlTabs, type BillingControlTab } from "../billing-control-tabs";
import { AutomationTab } from "./automation-tab";
import { CollectionsTab } from "./collections-tab";
import { ExceptionsTab } from "./exceptions-tab";
import { InvoicesTab } from "./invoices-tab";
import { RatingTab } from "./rating-tab";

/**
 * Billing control: one form, saved once, split into a tab for each job a person comes here
 * to do: what invoices say, how work moves to them on its own, what failed checks do, how
 * shipments are priced, and how overdue invoices are chased.
 */
export default function BillingControlForm() {
  const { data } = useSuspenseQuery({ ...billingControl.get() });
  const tabs = useBillingControlTabs();

  return (
    <SettingsForm<BillingControl, BillingControlTab>
      values={data}
      schema={billingControlSchema}
      queryKey={billingControl.get._def}
      save={(values) => billingControlService.update(values)}
      resourceName="Billing Control"
      prepare={prepareBillingControlForSubmit}
      tabs={tabs}
      content={(tab, { selectTab }) => {
        switch (tab) {
          case "invoices":
            return <InvoicesTab />;
          case "automation":
            return <AutomationTab />;
          case "exceptions":
            return <ExceptionsTab />;
          case "rating":
            return <RatingTab />;
          case "collections":
            return <CollectionsTab onOpenAutomation={() => selectTab("automation")} />;
        }
      }}
    />
  );
}
