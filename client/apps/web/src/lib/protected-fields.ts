import { Resource } from "@trenova/shared/types/generated/permission-resources";

import type { ProtectedFieldGroup } from "@trenova/shared/lib/reference-access";

export type { ProtectedFieldGroup };

export const PROTECTED_FIELD_GROUP_RESOURCES = {
  Personal: Resource.ProtectedPersonal,
  Margins: Resource.ProtectedMargins,
} as const satisfies Record<ProtectedFieldGroup, string>;
