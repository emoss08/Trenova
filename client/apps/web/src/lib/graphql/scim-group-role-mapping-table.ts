import { ScimGroupRoleMappingsTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export function createSCIMGroupRoleMappingTableGraphQLConfig(directoryId: string) {
  return defineDataTableGraphQLConfig({
    document: ScimGroupRoleMappingsTableDocument,
    operationName: "SCIMGroupRoleMappingsTable",
    connectionKey: "scimGroupRoleMappings",
    extraVariables: { directoryId },
  });
}

export type SCIMGroupRoleMappingRow = DataTableConfigRow<
  ReturnType<typeof createSCIMGroupRoleMappingTableGraphQLConfig>
>;
