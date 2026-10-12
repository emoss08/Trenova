import { AiUsageFeaturesDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

/** The key a slice with no feature goes by: calls made before features were kept. */
export const UNKNOWN_FEATURE_ID = "other";

/** Usage by feature over a window, as the shared data table reads it. */
export function createAIUsageFeaturesTableConfig(days: number) {
  return defineDataTableGraphQLConfig({
    document: AiUsageFeaturesDocument,
    operationName: "AIUsageFeatures",
    connectionKey: "aiUsageFeatures",
    extraVariables: { days },
    mapNode: (node) => ({ ...node, id: node.feature ?? UNKNOWN_FEATURE_ID }),
  });
}

export type AIUsageFeatureRow = DataTableConfigRow<
  ReturnType<typeof createAIUsageFeaturesTableConfig>
>;
