import { SettingsForm } from "@/components/settings/settings-form";
import { invoiceAdjustmentControl } from "@/lib/queries/invoice-adjustment-control";
import { invoiceAdjustmentControlService } from "@/services/invoice-adjustment-control";
import {
  invoiceAdjustmentControlFormSchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { prepareInvoiceAdjustmentControl } from "../invoice-adjustment-control-submit";
import {
  useInvoiceAdjustmentControlTabs,
  type InvoiceAdjustmentControlTab,
} from "../invoice-adjustment-control-tabs";
import { ApprovalTab } from "./approval-tab";
import { CreditsTab } from "./credits-tab";
import { EligibilityTab } from "./eligibility-tab";

/**
 * Invoice adjustment control: one form, saved once, split into a tab for each job a person
 * comes here to do: which invoices can be adjusted, who approves, and how credits and
 * rebills land.
 */
export default function InvoiceAdjustmentControlForm() {
  const { data } = useSuspenseQuery({ ...invoiceAdjustmentControl.get() });
  const tabs = useInvoiceAdjustmentControlTabs();

  return (
    <SettingsForm<InvoiceAdjustmentControl, InvoiceAdjustmentControlTab>
      values={data}
      schema={invoiceAdjustmentControlFormSchema}
      queryKey={invoiceAdjustmentControl.get._def}
      save={(values) => invoiceAdjustmentControlService.update(values)}
      resourceName="Invoice Adjustment Control"
      prepare={prepareInvoiceAdjustmentControl}
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "eligibility":
            return <EligibilityTab />;
          case "approval":
            return <ApprovalTab />;
          case "credits":
            return <CreditsTab />;
        }
      }}
    />
  );
}
