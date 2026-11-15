import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest, UnavailableFeature } from "@trenova/shared/types/permission";
import { permissionManifest } from "./permissions";

type AddOn = { addOn: string; addOnName: string; features: Record<string, string> };

/**
 * Every add-on and the features it carries, as the server's catalogue names them
 * (`platformcatalog/registry.go`). A base-only plan leaves out every one of these features.
 */
const ADD_ONS: readonly AddOn[] = [
  {
    addOn: "AI",
    addOnName: "Trenova AI",
    features: {
      "ai.desk": "The Desk",
      "ai.watchtower": "Watchtower",
      "ai.triage": "Inbox triage",
      "platform.agent_automation": "Agents",
      "ai.audit_trail": "AI audit trail",
    },
  },
  {
    addOn: "DOCI",
    addOnName: "Document Intelligence",
    features: { "document_intelligence.ai": "Document reading" },
  },
  {
    addOn: "DISPATCH",
    addOnName: "Dispatch Intelligence",
    features: { "tms.dispatch_intelligence": "Dispatch planner" },
  },
  {
    addOn: "ACC",
    addOnName: "Accounting & Cash",
    features: { "tms.accounting": "Accounting", "tms.factoring": "Factoring" },
  },
  {
    addOn: "RATE",
    addOnName: "Rating",
    features: { "tms.rating": "Rating", "tms.detention": "Detention" },
  },
  {
    addOn: "BRK",
    addOnName: "Carrier & Brokerage",
    features: { "tms.carrier_brokerage": "Carrier & brokerage" },
  },
  { addOn: "EDI", addOnName: "EDI", features: { "integrations.edi": "EDI" } },
  {
    addOn: "VIS",
    addOnName: "Visibility",
    features: {
      "tms.customer_tracking": "Customer tracking",
      "integrations.samsara": "Samsara",
      "integrations.google_maps": "Google Maps",
      "platform.table_change_alerts": "Data alerts",
    },
  },
  {
    addOn: "PAY",
    addOnName: "Driver Pay",
    features: { "tms.settlement": "Driver settlements", "tms.driver_portal": "Driver app" },
  },
  {
    addOn: "HR",
    addOnName: "HR",
    features: {
      "workforce.time_off": "Time off",
      "workforce.time_tracking": "Time tracking",
      "workforce.talent": "Training",
      "workforce.benefits": "Benefits",
      "workforce.self_service": "Self service",
    },
  },
  {
    addOn: "SAFE",
    addOnName: "Safety",
    features: { "workforce.compliance": "Compliance", "workforce.safety": "Safety" },
  },
  { addOn: "FUEL", addOnName: "Fuel", features: { "tms.fuel_tax": "Fuel tax" } },
  { addOn: "SEC", addOnName: "Security", features: { "platform.identity": "Single sign-on" } },
];

const ALL: readonly UnavailableFeature[] = ADD_ONS.flatMap(({ addOn, addOnName, features }) =>
  Object.entries(features).map(([feature, featureName]) => ({
    feature,
    featureName,
    addOn,
    addOnName,
  })),
);

/** What the manifest says a base-only plan leaves out: every add-on feature. */
export const BASE_ONLY_UNAVAILABLE: readonly UnavailableFeature[] = ALL;

/** What the manifest says a plan leaves out when it lacks exactly these features. */
export function unavailableFeatures(...features: string[]): UnavailableFeature[] {
  return ALL.filter((entry) => features.includes(entry.feature));
}

/** A signed-in person's manifest with these grants, on a plan that leaves out these features. */
export function planManifest(
  permissions: Record<string, number>,
  unavailable: readonly UnavailableFeature[],
): PermissionManifest {
  return { ...permissionManifest(permissions), unavailableFeatures: [...unavailable] };
}

/** Signs a test's person in with these grants on a plan that leaves out these features. */
export function seedPlan(
  permissions: Record<string, number>,
  unavailable: readonly UnavailableFeature[],
): void {
  usePermissionStore.setState({
    manifest: planManifest(permissions, unavailable),
    lastFetched: Date.now(),
    isLoading: false,
  });
}
