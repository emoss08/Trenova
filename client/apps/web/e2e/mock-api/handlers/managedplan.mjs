// The "Your plan" page's operation (`/admin/plan`, `managedPlan`), in the shape of
// services/tms/internal/api/graphql/schema/managed_plan.graphqls. The page exists only on a
// managed instance, so the public config says `managed` whenever the `managedPlan` scenario is not
// `selfHosted`: `managed` is the approved mockup's plan (usage, an add-on ending, the account link),
// `bare` the same plan before the control plane has sent a usage state or the account link,
// `ending` the managed plan set to end at the close of its period (the notice with Keep plan, and a
// final backup kept 7 days rather than 30), and
// `endingBare` the same end seen without the account link (the notice with no button).

const DAY = 86400;
const GIB = 1024 ** 3;
const MANAGE_URL =
  "https://console.trenova.cloud/instances/0193a1b2-0000-7000-8000-000000000001/plan";

const feature = (key, name) => ({ key, name });

const ADD_ONS = [
  {
    key: "AI",
    name: "Trenova AI",
    features: [
      feature("ai.desk", "Desk"),
      feature("ai.watchtower", "Watchtower"),
      feature("ai.triage", "Inbound triage"),
      feature("platform.agent_automation", "Agents"),
      feature("ai.audit_trail", "AI audit trail"),
    ],
  },
  {
    key: "ACC",
    name: "Accounting & Cash",
    features: [feature("tms.accounting", "Accounting"), feature("tms.factoring", "Factoring")],
  },
  {
    key: "VIS",
    name: "Visibility & Tracking",
    features: [
      feature("platform.table_change_alerts", "Change alerts"),
      feature("integrations.samsara", "Samsara"),
      feature("integrations.google_maps", "Google Maps"),
      feature("tms.customer_tracking", "Customer tracking"),
    ],
  },
  {
    key: "PAY",
    name: "Driver Pay",
    features: [feature("tms.settlement", "Driver settlements"), feature("dash.portal", "Driver portal")],
  },
  {
    key: "RTE",
    name: "Rating & Revenue Recovery",
    features: [feature("tms.rating", "Rating"), feature("tms.detention", "Detention")],
  },
  {
    key: "BRK",
    name: "Carrier & Brokerage",
    features: [feature("tms.carrier_brokerage", "Carrier brokerage")],
  },
  { key: "EDI", name: "EDI & Integrations", features: [feature("integrations.edi", "EDI")] },
  {
    key: "DSP",
    name: "Dispatch Intelligence",
    features: [feature("tms.dispatch_intelligence", "Dispatch intelligence")],
  },
  {
    key: "HRS",
    name: "HR Suite",
    features: [
      feature("workforce.time_off", "Time off"),
      feature("workforce.time_tracking", "Time tracking"),
      feature("workforce.talent", "Training and performance"),
      feature("workforce.benefits", "Benefits"),
      feature("workforce.self_service", "Employee self-service"),
    ],
  },
  {
    key: "SEC",
    name: "Security & Identity",
    features: [feature("platform.identity", "Single sign-on and provisioning")],
  },
  {
    key: "SAF",
    name: "Safety & Compliance",
    features: [
      feature("workforce.compliance", "Driver compliance"),
      feature("workforce.safety", "Safety events"),
    ],
  },
  { key: "FUL", name: "Fuel & IFTA", features: [feature("tms.fuel_tax", "Fuel and IFTA")] },
  {
    key: "DOC",
    name: "Document Intelligence",
    features: [feature("tms.document_intelligence", "Document intelligence")],
  },
];

const BASE = [
  feature("tms.core", "Freight"),
  feature("tms.dispatch", "Dispatch"),
  feature("tms.fleet", "Fleet"),
  feature("tms.billing", "Billing"),
  feature("workforce.core", "People"),
  feature("document_management", "Documents"),
  feature("analytics.workspace", "Reports and dashboards"),
  feature("platform.global_search", "Search"),
  feature("platform.realtime_notifications", "Notifications"),
  feature("integrations.exchange_rates", "Exchange rates"),
  feature("platform.api_keys", "API keys"),
  feature("platform.administration", "Administration"),
];

const ON_PLAN = new Set(["AI", "ACC", "VIS"]);

const addOnRef = (key) => {
  const addOn = ADD_ONS.find((entry) => entry.key === key);
  return { key: addOn.key, name: addOn.name };
};

function managedPlanOf(state) {
  const variant = state.scenario.managedPlan;
  const start = state.anchor - (state.anchor % DAY) - DAY;
  const end = start + 30 * DAY;
  const bare = variant === "bare" || variant === "endingBare";
  const ending = variant === "ending" || variant === "endingBare";
  return {
    managed: true,
    manageUrl: bare ? null : MANAGE_URL,
    endsAt: ending ? end : null,
    finalBackupDays: variant === "ending" ? 7 : 30,
    usage: bare
      ? null
      : {
          periodStart: start,
          periodEnd: end,
          meters: [
            {
              key: "shipments.total",
              name: "Shipments",
              unit: "COUNT",
              used: 312,
              included: 1500,
              hardCap: false,
              stopped: false,
            },
            {
              key: "documents.storage_bytes",
              name: "Document storage",
              unit: "BYTES",
              used: Math.round(18.4 * GIB),
              included: 100 * GIB,
              hardCap: false,
              stopped: false,
            },
            {
              key: "organizations.total",
              name: "Organizations",
              unit: "COUNT",
              used: 1,
              included: 1,
              hardCap: true,
              stopped: false,
            },
          ],
        },
    base: BASE,
    included: ADD_ONS.filter((addOn) => ON_PLAN.has(addOn.key)).map((addOn) => ({
      ...addOn,
      endsAt: addOn.key === "ACC" ? end : null,
    })),
    available: ADD_ONS.filter((addOn) => !ON_PLAN.has(addOn.key)).map((addOn) => ({
      ...addOn,
      endsAt: null,
    })),
    changes: [
      {
        changedAt: start + DAY,
        starting: false,
        left: [],
        joined: [],
        ending: [{ endsAt: end, addOn: addOnRef("ACC") }],
        kept: [],
      },
      {
        changedAt: start,
        starting: false,
        left: [],
        joined: [addOnRef("VIS")],
        ending: [],
        kept: [],
      },
      {
        changedAt: start,
        starting: true,
        left: [],
        joined: [addOnRef("AI"), addOnRef("ACC")],
        ending: [],
        kept: [],
      },
    ],
  };
}

const SELF_HOSTED = {
  managed: false,
  manageUrl: null,
  endsAt: null,
  finalBackupDays: 30,
  usage: null,
  base: [],
  included: [],
  available: [],
  changes: [],
};

const MANAGED_VARIANTS = new Set(["managed", "bare", "ending", "endingBare"]);

export function isManaged(state) {
  return MANAGED_VARIANTS.has(state.scenario.managedPlan);
}

export const MANAGED_PLAN_HANDLERS = {
  ManagedPlan: (state) => ({
    managedPlan: isManaged(state) ? managedPlanOf(state) : SELF_HOSTED,
  }),
};
