import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const RESOURCES_FILE = fileURLToPath(
  new URL("../../../../../packages/shared/src/types/generated/permission-resources.ts", import.meta.url),
);

const ALL_OPERATIONS = 0x3ffffff;

function resourceValues() {
  const source = readFileSync(RESOURCES_FILE, "utf8");
  return [...source.matchAll(/^\s+[A-Za-z]+: "([a-z_]+)",$/gm)].map((match) => match[1]);
}

export const ORG = { id: "org_mock", name: "Trenova Logistics" };

export const USER = {
  id: "usr_mock",
  version: 1,
  createdAt: 1700000000,
  updatedAt: 1700000000,
  businessUnitId: "bu_mock",
  currentOrganizationId: ORG.id,
  status: "Active",
  name: "Sam Admin",
  username: "sam",
  emailAddress: "sam@trenova.test",
  profilePicUrl: "",
  thumbnailUrl: "",
  timezone: "UTC",
  timeFormat: "24-hour",
  locale: "en",
  isLocked: false,
  mustChangePassword: false,
  lastLoginAt: 1700000000,
  assignments: [],
  memberships: [],
};

export function permissionManifest() {
  const permissions = Object.fromEntries(resourceValues().map((r) => [r, ALL_OPERATIONS]));
  return {
    version: "mock",
    userId: USER.id,
    organizationId: ORG.id,
    activeRoleIds: ["rol_admin"],
    authorizedRoleIds: ["rol_admin"],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
    maxSensitivity: "confidential",
    permissions,
    routeAccess: {},
    availableOrgs: [ORG],
    checksum: "mock",
    expiresAt: Math.floor(Date.now() / 1000) + 86400,
  };
}
