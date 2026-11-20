import { SettingsForm } from "@/components/settings/settings-form";
import { dispatchControl } from "@/lib/queries/dispatch-control";
import { dispatchControlService } from "@/services/dispatch-control";
import { dispatchControlSchema, type DispatchControl } from "@/types/dispatch-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useDispatchControlTabs, type DispatchControlTab } from "../dispatch-control-tabs";
import { AssignmentTab } from "./assignment-tab";
import { ComplianceTab } from "./compliance-tab";
import { CostsTab } from "./costs-tab";
import { ExceptionsTab } from "./exceptions-tab";
import { LearningTab } from "./learning-tab";
import { MonitoringTab } from "./monitoring-tab";
import { TendersTab } from "./tenders-tab";

/**
 * Dispatch control: one form, saved once, split into a tab for each job a person comes here
 * to do. Which tabs there are follows what the organization does: planning, its costs, what
 * it has learned and DOT compliance for one that runs a fleet, carrier tenders for a brokerage.
 */
export default function DispatchControlForm() {
  const { data } = useSuspenseQuery({ ...dispatchControl.get() });
  const tabs = useDispatchControlTabs();

  return (
    <SettingsForm<DispatchControl, DispatchControlTab>
      values={data}
      schema={dispatchControlSchema}
      queryKey={dispatchControl.get._def}
      save={(values) => dispatchControlService.update(values)}
      resourceName="Dispatch Control"
      tabs={tabs}
      content={(tab, { openTab, selectTab }) => {
        switch (tab) {
          case "assignment":
            return <AssignmentTab />;
          case "costs":
            return <CostsTab />;
          case "learning":
            return (
              <LearningTab
                shown={openTab === "learning"}
                onOpenAssignment={() => selectTab("assignment")}
              />
            );
          case "monitoring":
            return <MonitoringTab />;
          case "exceptions":
            return <ExceptionsTab />;
          case "tenders":
            return <TendersTab />;
          case "compliance":
            return <ComplianceTab />;
        }
      }}
    />
  );
}
