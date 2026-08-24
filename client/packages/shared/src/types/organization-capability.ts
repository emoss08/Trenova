/**
 * Organization capabilities describe which halves of the product an
 * organization actually operates: brokered freight, its own assets, or both.
 *
 * Permissions remain the real access control: a capability says whether the
 * organization uses that half of the product at all, never whether this user
 * may act on it. What a capability does beyond hiding UI is deliberately
 * asymmetric between the two:
 *
 * - `Brokerage` is hide-only. Every carrier-facing surface disappears, but the
 *   API stays open — turning it off declutters, it does not restrict anyone.
 * - `AssetOperations` is enforced. Driver assignment is refused server-side when
 *   it is off, because assigning a driver you do not employ corrupts data rather
 *   than merely cluttering a screen. Unassigning stays allowed, so an
 *   organization that inherits driver assignments can still unwind them. The
 *   client gating below therefore pre-empts a refusal the server will make
 *   anyway, instead of being the only thing standing in the way.
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
 * projection that predates the flags — resolves to this, so a gate can never be
 * the reason a feature disappears for an organization whose capabilities simply
 * are not known yet. The server enforces the asset half on its own record, so
 * failing open here never widens what an organization can actually do.
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
