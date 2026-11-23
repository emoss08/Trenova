/**
 * Feature keys the client names where a page, navigation entry or panel belongs to a feature
 * that has no permission resource of its own (a resource's feature comes from the server's
 * permission catalogue instead). They are the server's feature catalog keys and are never
 * shown: whether a feature is on the plan, and what it and its add-on are called, come from
 * the permission manifest's `unavailableFeatures` (`usePlanFeature`, `planFeatureOf`).
 */
export const PlanFeature = {
  Accounting: "tms.accounting",
  Administration: "platform.administration",
  APIKeys: "platform.api_keys",
  AgentAutomation: "platform.agent_automation",
  AIAuditTrail: "ai.audit_trail",
  AIDesk: "ai.desk",
  AITriage: "ai.triage",
  AIWatchtower: "ai.watchtower",
  CarrierBrokerage: "tms.carrier_brokerage",
  CustomerTracking: "tms.customer_tracking",
  Detention: "tms.detention",
  DispatchIntelligence: "tms.dispatch_intelligence",
  DocumentIntelligence: "document_intelligence.ai",
  DriverPay: "tms.settlement",
  DriverPortal: "tms.driver_portal",
  EDI: "integrations.edi",
  Factoring: "tms.factoring",
  GoogleMaps: "integrations.google_maps",
  Identity: "platform.identity",
  Rating: "tms.rating",
  Samsara: "integrations.samsara",
  TableChangeAlerts: "platform.table_change_alerts",
  WorkforceBenefits: "workforce.benefits",
  WorkforceCompliance: "workforce.compliance",
  WorkforceSafety: "workforce.safety",
  WorkforceTalent: "workforce.talent",
  WorkforceTimeOff: "workforce.time_off",
  WorkforceTimeTracking: "workforce.time_tracking",
} as const;

export type PlanFeatureKey = (typeof PlanFeature)[keyof typeof PlanFeature];
