import { Resource } from "@trenova/shared/types/generated/permission-resources";

export type ProtectedFieldGroup = "Personal" | "Margins";

export const PROTECTED_FIELD_GROUP_RESOURCES = {
  Personal: Resource.ProtectedPersonal,
  Margins: Resource.ProtectedMargins,
} as const satisfies Record<ProtectedFieldGroup, string>;
