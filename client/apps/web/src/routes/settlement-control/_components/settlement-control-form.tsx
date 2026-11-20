import { SettingsForm } from "@/components/settings/settings-form";
import { updateSettlementControl } from "@/lib/graphql/driver-settlement";
import { settlementControl } from "@/lib/queries/settlement-control";
import { settlementControlSchema, type SettlementControl } from "@/types/settlement-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useSettlementControlTabs, type SettlementControlTab } from "../settlement-control-tabs";
import { DeductionsTab } from "./deductions-tab";
import { PayPeriodTab } from "./pay-period-tab";
import { ReviewTab } from "./review-tab";

/**
 * Settlement control: one form, saved once, split into a tab for each job a person comes
 * here to do: the pay period, review and approval, and advances and escrow.
 */
export default function SettlementControlForm() {
  const { data } = useSuspenseQuery({ ...settlementControl.get() });
  const tabs = useSettlementControlTabs();

  return (
    <SettingsForm<SettlementControl, SettlementControlTab>
      values={data}
      schema={settlementControlSchema}
      queryKey={settlementControl.get._def}
      save={updateSettlementControl}
      resourceName="Settlement Control"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "payPeriod":
            return <PayPeriodTab />;
          case "review":
            return <ReviewTab />;
          case "deductions":
            return <DeductionsTab />;
        }
      }}
    />
  );
}
