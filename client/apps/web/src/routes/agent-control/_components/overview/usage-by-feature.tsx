import { DataTable } from "@/components/data-table/data-table";
import {
  createAIUsageFeaturesTableConfig,
  type AIUsageFeatureRow,
} from "@/lib/graphql/ai-usage-features-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { SecH } from "../kit/layout";
import { getUsageFeatureColumns } from "./usage-feature-columns";

type UsageByFeatureProps = {
  days: number;
  /** The busiest feature's calls, so every row's bar is drawn to one scale. */
  busiest: number;
};

/** Where the window's model calls went, by the feature that made them. */
export function UsageByFeature({ days, busiest }: UsageByFeatureProps) {
  const t = useT();
  const config = useMemo(() => createAIUsageFeaturesTableConfig(days), [days]);
  const columns = useMemo(() => getUsageFeatureColumns(t, busiest), [t, busiest]);

  return (
    <section className="sec">
      <SecH t={t("Usage by feature")} n={t("{0} days", days)} />
      <DataTable<AIUsageFeatureRow>
        name="AI usage by feature"
        queryKey="ai-usage-features"
        graphql={config}
        columns={columns}
        initialPageSize={10}
        enableExport={false}
      />
    </section>
  );
}
