import { SettingsForm } from "@/components/settings/settings-form";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { usePermission } from "@/hooks/use-permission";
import { dataRetention } from "@/lib/queries/data-retention";
import { updateDataRetention } from "@/services/data-retention";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  dataRetentionFormSchema,
  dataRetentionFormValues,
  type DataRetentionFormValues,
} from "../data-retention-form";
import { useDataRetentionTabs, type DataRetentionTab } from "../data-retention-tabs";
import { RetentionTab } from "./retention-tab";

/**
 * Data retention: one form, saved once, with one job, so one tab and no tab row. The form
 * holds the periods the server takes, and a save resets it to what the server kept, read back
 * through the same mapping as the first read.
 */
export default function DataRetentionForm({ now: nowProp }: { now?: number }) {
  const clock = useNowSeconds();
  const now = nowProp ?? clock;
  const { allowed: canUpdate } = usePermission(Resource.Organization, Operation.Update);
  const { data } = useSuspenseQuery(dataRetention.get());
  const tabs = useDataRetentionTabs();

  return (
    <SettingsForm<DataRetentionFormValues, DataRetentionTab>
      values={dataRetentionFormValues(data)}
      schema={dataRetentionFormSchema}
      queryKey={dataRetention.get().queryKey}
      save={async (values) => dataRetentionFormValues(await updateDataRetention(values))}
      resourceName="Data Retention"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "retention":
            return <RetentionTab now={now} readOnly={!canUpdate} />;
        }
      }}
    />
  );
}
