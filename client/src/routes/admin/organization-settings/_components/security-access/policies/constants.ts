import type { SelectOption } from "@/types/fields";

export const effectFilterOptions = [
  { value: "all", label: "All effects" },
  { value: "allow", label: "Allow" },
  { value: "deny", label: "Deny" },
] as const;

export const policyEffectOptions: SelectOption[] = [
  {
    value: "deny",
    label: "Deny",
    description: "Block matching access requests before lower-priority allow policies apply.",
  },
  {
    value: "allow",
    label: "Allow",
    description: "Permit matching access requests when no higher-priority deny policy applies.",
  },
];

export function accessPolicyPanelQueryKey(organizationId: string) {
  return `access-policy-list:${organizationId}`;
}
