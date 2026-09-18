import type { SelectOption } from "@trenova/shared/types/fields";

export const effectFilterOptions: SelectOption[] = [
  { value: "all", label: "All effects" },
  { value: "allow", label: "Allow", color: "var(--success)" },
  { value: "deny", label: "Deny", color: "var(--danger)" },
];

export const policyEffectOptions: SelectOption[] = [
  {
    value: "deny",
    label: "Deny",
    description: "Block matching access requests before lower-priority allow policies apply.",
    color: "var(--danger)",
  },
  {
    value: "allow",
    label: "Allow",
    description: "Permit matching access requests when no higher-priority deny policy applies.",
    color: "var(--success)",
  },
];

export function accessPolicyPanelQueryKey(organizationId: string) {
  return `access-policy-list:${organizationId}`;
}
