import {
  ScimGroupRoleMappingsTableDocument,
  type ScimGroupRoleMappingsTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { SCIMGroupRoleMapping } from "@trenova/shared/types/iam";

export function createSCIMGroupRoleMappingTableGraphQLConfig(directoryId: string) {
  return defineDataTableGraphQLConfig<
    SCIMGroupRoleMapping,
    ScimGroupRoleMappingsTableQueryVariables
  >({
    document: ScimGroupRoleMappingsTableDocument,
    operationName: "SCIMGroupRoleMappingsTable",
    connectionKey: "scimGroupRoleMappings",
    extraVariables: { directoryId },
  });
}
