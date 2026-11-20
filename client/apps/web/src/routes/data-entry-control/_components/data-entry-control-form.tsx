import { SettingsForm } from "@/components/settings/settings-form";
import { dataEntryControl } from "@/lib/queries/data-entry-control";
import { dataEntryControlService } from "@/services/data-entry-control";
import { dataEntryControlSchema, type DataEntryControl } from "@/types/data-entry-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useDataEntryControlTabs, type DataEntryControlTab } from "../data-entry-control-tabs";
import { FormattingTab } from "./formatting-tab";

/** Data entry control: one form, saved once, with one job, so one tab and no tab row. */
export default function DataEntryControlForm() {
  const { data } = useSuspenseQuery({ ...dataEntryControl.get() });
  const tabs = useDataEntryControlTabs();

  return (
    <SettingsForm<DataEntryControl, DataEntryControlTab>
      values={data}
      schema={dataEntryControlSchema}
      queryKey={dataEntryControl.get._def}
      save={(values) => dataEntryControlService.update(values)}
      resourceName="Data Entry Control"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "formatting":
            return <FormattingTab />;
        }
      }}
    />
  );
}
