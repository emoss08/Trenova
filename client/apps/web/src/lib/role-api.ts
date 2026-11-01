import {
  AssignRoleDocument,
  CreateRoleDocument,
  DeleteRoleDocument,
  PermissionBundlesDocument,
  PermissionOperationsDocument,
  PermissionResourcesDocument,
  RoleEditorDocument,
  RoleImpactDocument,
  RoleOptionsDocument,
  UnassignRoleDocument,
  UpdateRoleDocument,
  type RoleInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  roleSchema,
  userRoleAssignmentSchema,
  type AddPermission,
  type AssignRole,
  type DataScope,
  type FieldSensitivity,
  type Operation,
  type Role,
  type RoleImpact,
  type UserRoleAssignment,
} from "@trenova/shared/types/role";

type RequestOptions = { signal?: AbortSignal };

export type OperationDefinition = {
  operation: string;
  displayName: string;
  description: string;
  implies?: string[] | null;
};

export type ResourceDefinition = {
  resource: string;
  displayName: string;
  description: string;
  category: string;
  operations: OperationDefinition[];
  parentResource?: string;
  defaultSensitivity: string;
  enforceableScopes?: DataScope[] | null;
};

export type ResourceCategory = {
  category: string;
  resources: ResourceDefinition[];
};

type GraphQLResourceCategory = {
  category: string;
  resources: Array<{
    resource: string;
    displayName: string;
    description: string;
    category: string;
    operations: OperationDefinition[];
    parentResource?: string | null;
    defaultSensitivity: string;
    enforceableScopes: string[];
  }>;
};

export function resourceCategoriesFromGraphQL(
  categories: readonly GraphQLResourceCategory[],
): ResourceCategory[] {
  return categories.map((category) => ({
    category: category.category,
    resources: category.resources.map((resource) => ({
      resource: resource.resource,
      displayName: resource.displayName,
      description: resource.description,
      category: resource.category,
      operations: resource.operations.map((operation) => ({ ...operation })),
      ...(resource.parentResource ? { parentResource: resource.parentResource } : {}),
      defaultSensitivity: resource.defaultSensitivity,
      enforceableScopes: resource.enforceableScopes as DataScope[],
    })),
  }));
}

export async function getAvailableResources(
  options: RequestOptions = {},
): Promise<ResourceCategory[]> {
  const data = await requestGraphQL<{ permissionResources: GraphQLResourceCategory[] }>({
    document: PermissionResourcesDocument,
    operationName: "PermissionResources",
    signal: options.signal,
  });
  return resourceCategoriesFromGraphQL(data.permissionResources);
}

export type BundleGrant = {
  resource: string;
  displayName: string;
  operations: Operation[];
  dataScope: DataScope;
};

export type CapabilityBundle = {
  id: string;
  module: string;
  label: string;
  description: string;
  grants: BundleGrant[];
};

export type JobTemplate = {
  id: string;
  label: string;
  description: string;
  bundles: string[] | null;
  maxSensitivity: FieldSensitivity;
  grants: BundleGrant[];
};

export type BundleCatalog = {
  bundles: CapabilityBundle[];
  templates: JobTemplate[];
};

type GraphQLBundleGrant = {
  resource: string;
  displayName: string;
  operations: string[];
  dataScope: string;
};

function bundleGrantFromGraphQL(grant: GraphQLBundleGrant): BundleGrant {
  return {
    resource: grant.resource,
    displayName: grant.displayName,
    operations: grant.operations as Operation[],
    dataScope: grant.dataScope as DataScope,
  };
}

export async function getPermissionBundles(options: RequestOptions = {}): Promise<BundleCatalog> {
  const data = await requestGraphQL<{
    permissionBundles: {
      bundles: Array<Omit<CapabilityBundle, "grants"> & { grants: GraphQLBundleGrant[] }>;
      templates: Array<{
        id: string;
        label: string;
        description: string;
        bundles: string[];
        maxSensitivity: string;
        grants: GraphQLBundleGrant[];
      }>;
    };
  }>({
    document: PermissionBundlesDocument,
    operationName: "PermissionBundles",
    signal: options.signal,
  });
  return {
    bundles: data.permissionBundles.bundles.map((bundle) => ({
      id: bundle.id,
      module: bundle.module,
      label: bundle.label,
      description: bundle.description,
      grants: bundle.grants.map(bundleGrantFromGraphQL),
    })),
    templates: data.permissionBundles.templates.map((template) => ({
      id: template.id,
      label: template.label,
      description: template.description,
      bundles: template.bundles,
      maxSensitivity: template.maxSensitivity as FieldSensitivity,
      grants: template.grants.map(bundleGrantFromGraphQL),
    })),
  };
}

export async function getAvailableOperations(
  options: RequestOptions = {},
): Promise<OperationDefinition[]> {
  const data = await requestGraphQL({
    document: PermissionOperationsDocument,
    operationName: "PermissionOperations",
    signal: options.signal,
  });
  return data.permissionOperations.map((operation) => ({ ...operation }));
}

export type RoleOption = {
  id: string;
  name: string;
  description: string;
  maxSensitivity: string;
  isSystem: boolean;
};

/** The most roles one read of a picker asks for; the server serves at most 100 a page. */
const ROLE_OPTIONS_PAGE = 100;

export async function listRoles(options: RequestOptions = {}): Promise<RoleOption[]> {
  const data = await requestGraphQL({
    document: RoleOptionsDocument,
    operationName: "RoleOptions",
    variables: { first: ROLE_OPTIONS_PAGE },
    signal: options.signal,
  });
  return data.roles.edges.map((edge) => ({ ...edge.node }));
}

export async function getRole(id: string, options: RequestOptions = {}): Promise<Role> {
  const data = await requestGraphQL<{
    role: Record<string, unknown> | null;
    rolePermissions: unknown;
  }>({
    document: RoleEditorDocument,
    operationName: "RoleEditor",
    variables: { id },
    signal: options.signal,
  });
  if (!data.role) {
    throw new Error(`Role ${id} was not found`);
  }
  return safeParse(roleSchema, { ...data.role, permissions: data.rolePermissions }, "Role");
}

type RoleWrite = {
  name: string;
  description?: string;
  coreResponsibility?: string | null;
  maxSensitivity: FieldSensitivity;
  parentRoleIds?: string[] | null;
  permissions?: AddPermission[];
};

function roleInput(values: RoleWrite): RoleInput {
  return {
    name: values.name,
    description: values.description ?? "",
    coreResponsibility: values.coreResponsibility ?? null,
    maxSensitivity: values.maxSensitivity,
    parentRoleIds: values.parentRoleIds ?? null,
    permissions: values.permissions
      ? values.permissions.map((permission) => ({
          resource: permission.resource,
          operations: permission.operations,
          dataScope: permission.dataScope,
        }))
      : null,
  };
}

export async function createRole(values: RoleWrite): Promise<Role> {
  const data = await requestGraphQL<{ createRole: unknown }>({
    document: CreateRoleDocument,
    operationName: "CreateRole",
    variables: { input: roleInput(values) },
  });
  return safeParse(roleSchema, data.createRole, "Role");
}

export async function updateRole(id: string, values: RoleWrite): Promise<Role> {
  const data = await requestGraphQL<{ updateRole: unknown }>({
    document: UpdateRoleDocument,
    operationName: "UpdateRole",
    variables: { id, input: roleInput(values) },
  });
  return safeParse(roleSchema, data.updateRole, "Role");
}

export async function deleteRole(id: string, options?: { force?: boolean }): Promise<void> {
  await requestGraphQL({
    document: DeleteRoleDocument,
    operationName: "DeleteRole",
    variables: { id, force: options?.force ?? false },
  });
}

export async function getRoleImpact(
  id: string,
  options: RequestOptions = {},
): Promise<RoleImpact[]> {
  const data = await requestGraphQL({
    document: RoleImpactDocument,
    operationName: "RoleImpact",
    variables: { roleId: id },
    signal: options.signal,
  });
  return data.roleImpact.map((impact) => ({ ...impact }));
}

export async function assignRole(roleId: string, values: AssignRole): Promise<UserRoleAssignment> {
  const data = await requestGraphQL<{ assignRole: unknown }>({
    document: AssignRoleDocument,
    operationName: "AssignRole",
    variables: {
      roleId,
      input: { userId: values.userId, expiresAt: values.expiresAt ?? null },
    },
  });
  return safeParse(userRoleAssignmentSchema, data.assignRole, "Role assignment");
}

export async function unassignRole(assignmentId: string): Promise<void> {
  await requestGraphQL({
    document: UnassignRoleDocument,
    operationName: "UnassignRole",
    variables: { assignmentId },
  });
}
