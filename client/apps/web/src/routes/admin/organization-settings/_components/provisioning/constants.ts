import type { SCIMDirectory, SCIMGroupRoleMapping } from "@trenova/shared/types/iam";

export const emptyDirectory: SCIMDirectory = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  tenantSlug: "",
  enabled: true,
  createdAt: 0,
  updatedAt: 0,
};

export const emptyMapping: SCIMGroupRoleMapping = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  directoryId: "",
  externalGroupId: "",
  displayName: "",
  roleId: "",
  createdAt: 0,
  updatedAt: 0,
};

export function scimDirectoryPanelQueryKey(organizationId: string) {
  return `scim-directory-panel:${organizationId}`;
}

export function scimGroupMappingPanelQueryKey(organizationId: string, directoryId: string) {
  return `scim-group-mapping-panel:${organizationId}:${directoryId}`;
}
