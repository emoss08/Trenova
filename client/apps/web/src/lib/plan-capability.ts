/**
 * Capabilities a plan can withhold, as the server names them in the edition's
 * `restrictions`. Mirrors platformplan.Capability.
 */
export const PlanCapability = {
  EmailOutbound: "email.outbound",
  Integrations: "integrations",
  APIKeys: "api_keys",
  AgentAutomation: "agent.automation",
  AgentWebSearch: "agent.web_search",
  CarrierIntelligencePaid: "carrier_intelligence.paid",
  DocumentIntelligence: "document_intelligence",
  SMS: "sms",
  SSO: "sso",
} as const;

export type PlanCapabilityType = (typeof PlanCapability)[keyof typeof PlanCapability];

/**
 * Whether the organization's plan withholds the capability. No capability, or no
 * restrictions (self-hosted, unlimited plans), means nothing is withheld.
 */
export function isPlanRestricted(
  restrictions: readonly string[] | undefined,
  capability: PlanCapabilityType | undefined,
): boolean {
  if (!capability || !restrictions || restrictions.length === 0) {
    return false;
  }

  return restrictions.includes(capability);
}
