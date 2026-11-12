import type { DeskCase, DeskCaseDecision, DeskCaseOption } from "@/lib/graphql/desk-case";
import type { AssistantProposal } from "@/types/assistant";

const AGENT = {
  id: "agd_dispatch",
  name: "Dispatch desk",
  description: null,
  icon: "",
  accent: "teal",
  template: null,
};

/** The plan on a case as `DeskCaseDecisionFields` selects it. */
export function caseDecision(overrides: Partial<DeskCaseDecision> = {}): DeskCaseDecision {
  return {
    kind: "Plan",
    id: "apl_1",
    title: "Move SHP-1's delivery to 1:00 PM",
    phase: "Awaiting",
    agent: AGENT,
    proposedAt: 1_800_000_000,
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    commitsAt: null,
    decidedAt: null,
    decidedBy: null,
    note: "",
    itemId: "wt_1",
    others: 0,
    ...overrides,
  };
}

export function caseOption(overrides: Partial<DeskCaseOption> = {}): DeskCaseOption {
  return {
    id: "rco_1",
    kind: "Plan",
    label: "Move the delivery to 1:00 PM",
    outcome: "1:18 PM",
    outcomeNote: "inside the new window",
    tone: "Good",
    cost: "No fee if Acme confirms",
    description: "John keeps the load and delivers after his break.",
    recommended: true,
    chosen: true,
    ...overrides,
  };
}

/** A late shipment's case as `DeskCase` reads it: one finding, four exhibits, one plan. */
export function deskCase(overrides: Partial<DeskCase> = {}): DeskCase {
  return {
    away: null,
    ownership: {
      owner: null,
      auto: false,
      covering: false,
      canWork: true,
      canClaim: true,
      canAssign: false,
    },
    handOff: null,
    record: {
      kind: "shipment",
      noun: "shipment",
      id: "shp_1",
      label: "SHP-1",
      subtitle: "",
      status: "",
      readable: true,
    },
    head: {
      references: ["Acme Foods", "BOL 88213"],
      title: "Dallas, TX → Chicago, IL",
      status: "In transit · projected late",
      statusTone: "Bad",
      details: ["John Smith · TRC-001 · TRL-001", "Delivery 2:00–3:00 PM at Acme Foods DC"],
      deadline: { at: 1_800_003_600, label: "Delivery window closes" },
    },
    finding: {
      agent: AGENT,
      foundAt: 1_800_000_000,
      sources: 2,
      raised: "Watchtower raised this at 6:31 AM from eta at risk.",
      computed: "Projected to reach the delivery after its window closes.",
      readingStatus: "Ready",
      reading: [
        { text: "John Smith has ", highlight: false, cite: null },
        { text: "2h 10m of driving left", highlight: true, cite: null },
        { text: "", highlight: false, cite: 2 },
        { text: " and is 215 miles out", highlight: false, cite: null },
        { text: "", highlight: false, cite: 1 },
        { text: ".", highlight: false, cite: null },
      ],
      trust: [
        { key: "Matters", text: "Acme is staffing the door." },
        { key: "Unchanged", text: "Delivered about 7:05 PM." },
        { key: "Proposes", text: "John keeps the load and delivers after his break." },
        { key: "Approval", text: "A person approves it." },
      ],
    },
    options: [],
    drafting: false,
    draftNote: "",
    decision: caseDecision(),
    exhibits: [
      {
        key: "position",
        number: 1,
        kind: "Route",
        title: "Where it is",
        source: "Telematics · TRC-001",
        at: 1_800_000_000,
        read: true,
        route: {
          points: [
            { label: "Dallas", x: 0.06, y: 0.94, latitude: 32.78, longitude: -96.8, state: "Done" },
            {
              label: "Chicago",
              x: 0.94,
              y: 0.06,
              latitude: 41.88,
              longitude: -87.63,
              state: "Ahead",
            },
          ],
          truck: { x: 0.7, y: 0.3, latitude: 39.78, longitude: -89.65 },
          callout: "break due ~8:50 AM",
          calloutTone: "Warning",
        },
        meters: [],
        message: null,
        timeline: [],
        facts: [],
        note: "Springfield, IL · moving 61 mph · 215 miles to Chicago",
        path: "",
      },
      {
        key: "hours",
        number: 2,
        kind: "Meters",
        title: "John Smith's hours",
        source: "ELD · hours of service",
        at: 1_800_000_000,
        read: true,
        route: null,
        meters: [
          { label: "Driving left", value: "2h 10m of 11h", fraction: 0.2, tone: "Warning" },
          { label: "On duty left", value: "4h 05m of 14h", fraction: 0.29, tone: "Neutral" },
        ],
        message: null,
        timeline: [],
        facts: [],
        note: "Break due by 8:50 AM.",
        path: "",
      },
    ],
    note: "Watchtower: critical, raised 6:31 AM by eta at risk monitoring.",
    starters: ["What happens if nothing changes?"],
    open: 2,
    severity: "Critical",
    ...overrides,
  };
}

/** One step of a plan as the Desk's presenters read a proposal. */
export function planStep(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "shipment_reschedule_stop",
    arguments: { shipmentId: "shp_1", stopId: "stp_1", windowStart: "2026-11-10T13:00" },
    rationale: "The projected arrival is after the window closes.",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    sourceMessageId: "",
    confidence: 0.9,
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    hold: null,
    planId: "apl_1",
    planStep: 1,
    simulatedAt: null,
    simulation: null,
    fields: [],
    modifications: null,
    evidence: [],
    reviewRequired: false,
    ...overrides,
  } as AssistantProposal;
}
