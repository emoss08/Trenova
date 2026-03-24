import type {
  AddPermission,
  AssignRole,
  ResourcePermission,
  Role,
  RoleImpact,
  UserRoleAssignment,
} from "@/types/role";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { api } from "./api";

export type OperationDefinition = {
  operation: string;
  displayName: string;
  description: string;
};

export type ResourceDefinition = {
  resource: string;
  displayName: string;
  description: string;
  category: string;
  operations: OperationDefinition[];
  parentResource?: string;
  defaultSensitivity: string;
};

export type ResourceCategory = {
  category: string;
  resources: ResourceDefinition[];
};

export async function getAvailableResources(): Promise<ResourceCategory[]> {
  return api.get<ResourceCategory[]>("/permissions/resources");
}

export async function getAvailableOperations(): Promise<OperationDefinition[]> {
  return api.get<OperationDefinition[]>("/permissions/operations");
}

export type ListRolesParams = {
  limit?: number;
  offset?: number;
  query?: string;
  includeSystem?: boolean;
};

export async function listRoles(
  params?: ListRolesParams,
): Promise<GenericLimitOffsetResponse<Role>> {
  const searchParams = new URLSearchParams();

  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  if (params?.query) searchParams.set("query", params.query);
  if (params?.includeSystem) searchParams.set("includeSystem", "true");

  const queryString = searchParams.toString();
  const endpoint = `/roles/${queryString ? `?${queryString}` : ""}`;

  return api.get(endpoint);
}

export async function getRole(id: string): Promise<Role> {
  return api.get<Role>(`/roles/${id}`);
}

export async function createRole(
  data: Omit<
    Role,
    | "id"
    | "organizationId"
    | "createdBy"
    | "createdAt"
    | "updatedAt"
    | "permissions"
  >,
): Promise<Role> {
  return api.post<Role>("/roles/", data);
}

export async function updateRole(
  id: string,
  data: Partial<Role>,
): Promise<Role> {
  return api.put<Role>(`/roles/${id}`, data);
}

export async function deleteRole(id: string): Promise<void> {
  return api.delete(`/roles/${id}`);
}

export async function getRoleImpact(id: string): Promise<RoleImpact[]> {
  return api.get<RoleImpact[]>(`/roles/${id}/impact`);
}

export async function addPermission(
  roleId: string,
  data: AddPermission,
): Promise<ResourcePermission> {
  return api.post<ResourcePermission>(`/roles/${roleId}/permissions`, data);
}

export async function updatePermission(
  roleId: string,
  permissionId: string,
  data: Partial<AddPermission>,
): Promise<ResourcePermission> {
  return api.put<ResourcePermission>(
    `/roles/${roleId}/permissions/${permissionId}`,
    data,
  );
}

export async function removePermission(
  roleId: string,
  permissionId: string,
): Promise<void> {
  return api.delete(`/roles/${roleId}/permissions/${permissionId}`);
}

export async function assignRole(
  roleId: string,
  data: AssignRole,
): Promise<UserRoleAssignment> {
  return api.post<UserRoleAssignment>(`/roles/${roleId}/assignments`, data);
}

export async function unassignRole(assignmentId: string): Promise<void> {
  return api.delete(`/roles/assignments/${assignmentId}`);
}

export async function getRoleAssignments(
  roleId: string,
): Promise<UserRoleAssignment[]> {
  return api.get<UserRoleAssignment[]>(`/roles/${roleId}/assignments`);
}
