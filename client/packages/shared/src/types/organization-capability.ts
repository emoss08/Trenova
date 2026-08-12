/**
 * Organization capabilities describe which halves of the product an
 * organization actually operates: brokered freight, its own assets, or both.
 *
 * They gate navigation and route *visibility* only. Permissions remain the real
 * access control and the API stays open regardless of these flags — turning a
 * capability off declutters the UI, it does not restrict anyone.
 */
export const OrganizationCapability = {
  Brokerage: "brokerage",
  AssetOperations: "assetOperations",
} as const;

export type OrganizationCapabilityType =
  (typeof OrganizationCapability)[keyof typeof OrganizationCapability];

export interface OrganizationCapabilities {
  brokerageEnabled: boolean;
  assetOperationsEnabled: boolean;
}

/**
 * The fail-open baseline. Every unknown state — no session, no membership, a
 * projection that predates the flags — resolves to this, so a cosmetic gate can
 * never be the reason a feature disappears.
 */
export const ORGANIZATION_CAPABILITIES_ENABLED: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};

const ORGANIZATION_CAPABILITY_LABELS: Record<OrganizationCapabilityType, string> = {
  [OrganizationCapability.Brokerage]: "Brokerage",
  [OrganizationCapability.AssetOperations]: "Asset Operations",
};

export function organizationCapabilityLabel(capability: OrganizationCapabilityType): string {
  return ORGANIZATION_CAPABILITY_LABELS[capability];
}

export function hasOrganizationCapability(
  capabilities: OrganizationCapabilities,
  capability: OrganizationCapabilityType,
): boolean {
  return capability === OrganizationCapability.Brokerage
    ? capabilities.brokerageEnabled
    : capabilities.assetOperationsEnabled;
}

/**
 * The Asset / Brokerage / Hybrid question every operator actually asks, expressed
 * as a preset over the two independent flags. Gating code never reads the preset —
 * it exists so the settings screen can offer one decision instead of two, while
 * the switches underneath stay individually editable.
 */
export type OrganizationCapabilityPreset = "asset" | "brokerage" | "hybrid" | "custom";

const ORGANIZATION_CAPABILITY_PRESET_VALUES: Record<
  Exclude<OrganizationCapabilityPreset, "custom">,
  OrganizationCapabilities
> = {
  asset: { brokerageEnabled: false, assetOperationsEnabled: true },
  brokerage: { brokerageEnabled: true, assetOperationsEnabled: false },
  hybrid: { brokerageEnabled: true, assetOperationsEnabled: true },
};

export function organizationCapabilityPresetValues(
  preset: Exclude<OrganizationCapabilityPreset, "custom">,
): OrganizationCapabilities {
  return ORGANIZATION_CAPABILITY_PRESET_VALUES[preset];
}

export function resolveOrganizationCapabilityPreset(
  capabilities: OrganizationCapabilities,
): OrganizationCapabilityPreset {
  const match = Object.entries(ORGANIZATION_CAPABILITY_PRESET_VALUES).find(
    ([, values]) =>
      values.brokerageEnabled === capabilities.brokerageEnabled &&
      values.assetOperationsEnabled === capabilities.assetOperationsEnabled,
  );

  return (match?.[0] as OrganizationCapabilityPreset | undefined) ?? "custom";
}
