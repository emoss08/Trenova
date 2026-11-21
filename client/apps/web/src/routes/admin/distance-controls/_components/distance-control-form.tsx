import { SettingsForm } from "@/components/settings/settings-form";
import { distanceControl } from "@/lib/queries/distance-control";
import { distanceControlService } from "@/services/distance-control";
import { distanceControlFormSchema, type DistanceControl } from "@/types/distance-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useDistanceControlTabs, type DistanceControlTab } from "../distance-control-tabs";
import { MileageTab } from "./mileage-tab";

/** Distance control: one form, saved once, with one job, so one tab and no tab row. */
export default function DistanceControlForm() {
  const { data } = useSuspenseQuery({ ...distanceControl.get() });
  const tabs = useDistanceControlTabs();

  return (
    <SettingsForm<DistanceControl, DistanceControlTab>
      values={data}
      schema={distanceControlFormSchema}
      queryKey={distanceControl.get._def}
      save={(values) => distanceControlService.patch(values)}
      resourceName="Distance Control"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "mileage":
            return <MileageTab />;
        }
      }}
    />
  );
}
