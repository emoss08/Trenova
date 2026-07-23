import type { SelectOption } from "@trenova/shared/types/fields";

export const effectFilterOptions: SelectOption[] = [
  { value: "all", label: "All effects" },
  { value: "allow", label: "Allow", color: "#15803d" },
  { value: "deny", label: "Deny", color: "#dc2626" },
];

export const policyEffectOptions: SelectOption[] = [
  {
    value: "deny",
    label: "Deny",
    description: "Block matching access requests before lower-priority allow policies apply.",
    color: "#dc2626",
  },
  {
    value: "allow",
    label: "Allow",
    description: "Permit matching access requests when no higher-priority deny policy applies.",
    color: "#15803d",
  },
];

export function accessPolicyPanelQueryKey(organizationId: string) {
  return `access-policy-list:${organizationId}`;
}
