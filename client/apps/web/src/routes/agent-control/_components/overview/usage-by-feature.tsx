import { DataTable } from "@/components/data-table/data-table";
import { SectionPanel } from "@/components/section-panel";
import {
  createAIUsageFeaturesTableConfig,
  type AIUsageFeatureRow,
} from "@/lib/graphql/ai-usage-features-table";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import { useMemo, useState } from "react";
import { getUsageFeatureColumns } from "./usage-feature-columns";

const WITH_FAILURES: FieldFilter[] = [{ field: "failed", operator: "gt", value: 0 }];

type UsageByFeatureProps = {
  days: number;
  /** The busiest feature's calls, so every row's bar is drawn to one scale. */
  busiest: number;
};

/** Where the window's model calls went, by the feature that made them. */
export function UsageByFeature({ days, busiest }: UsageByFeatureProps) {
  const t = useT();
  const [failuresOnly, setFailuresOnly] = useState(false);
  const config = useMemo(() => createAIUsageFeaturesTableConfig(days), [days]);
  const columns = useMemo(() => getUsageFeatureColumns(t, busiest), [t, busiest]);

  return (
    <SectionPanel title={t("Usage by feature")} hint={t("{0} days", days)}>
      <DataTable<AIUsageFeatureRow>
        name="AI usage by feature"
        queryKey="ai-usage-features"
        graphql={config}
        columns={columns}
        initialPageSize={10}
        scopeFilters={failuresOnly ? WITH_FAILURES : undefined}
        toolbar={{
          trailing: (
            <Button
              size="sm"
              variant={failuresOnly ? "secondary" : "outline"}
              aria-pressed={failuresOnly}
              onClick={() => setFailuresOnly((on) => !on)}
            >
              {t("With failures")}
            </Button>
          ),
        }}
      />
    </SectionPanel>
  );
}
