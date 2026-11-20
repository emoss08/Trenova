import { SettingsForm } from "@/components/settings/settings-form";
import { updateCarrierSettlementControl } from "@/lib/graphql/carrier-settlement";
import { carrierSettlementControl } from "@/lib/queries/carrier-settlement-control";
import {
  carrierSettlementControlSchema,
  type CarrierSettlementControl,
} from "@/types/carrier-settlement-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  useCarrierSettlementControlTabs,
  type CarrierSettlementControlTab,
} from "../carrier-settlement-control-tabs";
import { ApprovalTab } from "./approval-tab";
import { MatchingTab } from "./matching-tab";
import { PayPeriodTab } from "./pay-period-tab";

/**
 * Carrier settlement control: one form, saved once, split into a tab for each job a person
 * comes here to do: the pay period, matching carrier invoices, and approving and posting.
 */
export default function CarrierSettlementControlForm() {
  const { data } = useSuspenseQuery({ ...carrierSettlementControl.get() });
  const tabs = useCarrierSettlementControlTabs();

  return (
    <SettingsForm<CarrierSettlementControl, CarrierSettlementControlTab>
      values={data}
      schema={carrierSettlementControlSchema}
      queryKey={carrierSettlementControl.get._def}
      save={updateCarrierSettlementControl}
      resourceName="Carrier Settlement Control"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "payPeriod":
            return <PayPeriodTab />;
          case "matching":
            return <MatchingTab />;
          case "approval":
            return <ApprovalTab />;
        }
      }}
    />
  );
}
