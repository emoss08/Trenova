import { SettingsForm } from "@/components/settings/settings-form";
import { shipmentControl } from "@/lib/queries/shipment-control";
import { shipmentControlService } from "@/services/shipment-control";
import { shipmentControlSchema, type ShipmentControl } from "@/types/shipment-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { sanitizeShipmentControlForSubmit } from "../shipment-control-submit";
import { useShipmentControlTabs, type ShipmentControlTab } from "../shipment-control-tabs";
import { AutomationTab } from "./automation-tab";
import { DetentionTab } from "./detention-tab";
import { EntryTab } from "./entry-tab";

/**
 * Shipment control: one form, saved once, split into a tab for each job a person comes here
 * to do. Detention is there only when the plan includes it.
 */
export default function ShipmentControlForm() {
  const { data } = useSuspenseQuery({ ...shipmentControl.get() });
  const tabs = useShipmentControlTabs();

  return (
    <SettingsForm<ShipmentControl, ShipmentControlTab>
      values={data}
      schema={shipmentControlSchema}
      queryKey={shipmentControl.get._def}
      save={(values) => shipmentControlService.update(values)}
      resourceName="Shipment Control"
      prepare={sanitizeShipmentControlForSubmit}
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "entry":
            return <EntryTab />;
          case "automation":
            return <AutomationTab />;
          case "detention":
            return <DetentionTab />;
        }
      }}
    />
  );
}
