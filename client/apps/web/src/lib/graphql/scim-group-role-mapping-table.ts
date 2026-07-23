import {
  ScimGroupRoleMappingsTableDocument,
  type ScimGroupRoleMappingsTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { SCIMGroupRoleMapping } from "@/types/iam";

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
