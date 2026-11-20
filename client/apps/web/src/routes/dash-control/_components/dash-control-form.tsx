import { SettingsForm } from "@/components/settings/settings-form";
import { updateDashControl } from "@/lib/graphql/dash-control";
import { dashControl } from "@/lib/queries/dash-control";
import { dashControlSchema, type DashControl } from "@/types/dash-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { prepareDashControlForSubmit } from "../dash-control-submit";
import { useDashControlTabs, type DashControlTab } from "../dash-control-tabs";
import { LoadsTab } from "./loads-tab";
import { PayTab } from "./pay-tab";
import { ProfileTab } from "./profile-tab";
import { RemindersTab } from "./reminders-tab";

/**
 * Dash control: what drivers can see and do in the driver app, in one form saved once and
 * split into a tab for each part of a driver's day: loads, pay, their record and reminders.
 */
export default function DashControlForm() {
  const { data } = useSuspenseQuery({ ...dashControl.get() });
  const tabs = useDashControlTabs();

  return (
    <SettingsForm<DashControl, DashControlTab>
      values={data}
      schema={dashControlSchema}
      queryKey={dashControl.get._def}
      save={updateDashControl}
      resourceName="Dash Control"
      prepare={prepareDashControlForSubmit}
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "loads":
            return <LoadsTab />;
          case "pay":
            return <PayTab />;
          case "profile":
            return <ProfileTab />;
          case "reminders":
            return <RemindersTab />;
        }
      }}
    />
  );
}
