import { parseAsArrayOf, parseAsString, parseAsStringLiteral } from "nuqs";

export const aiControlTabValues = [
  "overview",
  "agents",
  "providers",
  "extensions",
  "memory",
  "retrieval",
  "safety",
  "quality",
  "activity",
  "audit",
] as const;
export type AIControlTab = (typeof aiControlTabValues)[number];

export const AI_CONTROL_TAB_PARAM = "tab";

export const aiControlTabParser = parseAsStringLiteral(aiControlTabValues)
  .withOptions({ history: "push", shallow: true })
  .withDefault("overview");

export const activityViews = ["runs", "proposals", "plans", "evaluations", "exceptions"] as const;
export type ActivityView = (typeof activityViews)[number];

export const ACTIVITY_VIEW_PARAM = "activity";

export const activityViewParser = parseAsStringLiteral(activityViews)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("runs");

/** Safety is two tables: every tool's rule, and what picked agents make of the tools they hold. */
export const safetyViews = ["rules", "agents"] as const;
export type SafetyView = (typeof safetyViews)[number];

export const SAFETY_VIEW_PARAM = "safety";

export const safetyViewParser = parseAsStringLiteral(safetyViews)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("rules");

/** The agents compared under Safety, kept in the address so a comparison can be shared. */
export const SAFETY_AGENTS_PARAM = "safetyAgents";

export const safetyAgentsParser = parseAsArrayOf(parseAsString)
  .withOptions({ history: "replace", shallow: true })
  .withDefault([]);

/**
 * Quality is one table at a time: the agents, their suite runs (and a run's
 * cases), the answers people liked least, the golden set, and the sweep's
 * settings.
 */
export const qualityViews = ["agents", "runs", "ratings", "golden", "settings"] as const;
export type QualityView = (typeof qualityViews)[number];

export const QUALITY_VIEW_PARAM = "quality";

/** No default: a link that names only a suite run opens its cases under Suite runs. */
export const qualityViewParser = parseAsStringLiteral(qualityViews).withOptions({
  history: "replace",
  shallow: true,
});

/** Narrows suite runs and worst-rated answers to one agent; links from notifications set it. */
export const QUALITY_AGENT_PARAM = "agent";
/** Opens one suite run's cases in place of the runs. */
export const QUALITY_SUITE_RUN_PARAM = "suiteRun";

export const qualityAgentParser = parseAsString.withOptions({ history: "push", shallow: true });
export const qualitySuiteRunParser = parseAsString.withOptions({ history: "push", shallow: true });

/** Narrows Retrieval's failed items to one source; a source's row sets it. */
export const RETRIEVAL_SOURCE_PARAM = "retrievalSource";

export const retrievalSourceValues = ["Memory", "Document", "InboundMessage"] as const;
export type RetrievalSource = (typeof retrievalSourceValues)[number];

export const retrievalSourceParser = parseAsStringLiteral(retrievalSourceValues).withOptions({
  history: "replace",
  shallow: true,
});

/**
 * The audit trail is two tables: the signed trail of what agents did, and the
 * files it has been exported to.
 */
export const auditViews = ["trail", "exports"] as const;
export type AuditView = (typeof auditViews)[number];

export const AUDIT_VIEW_PARAM = "audit";

export const auditViewParser = parseAsStringLiteral(auditViews)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("trail");

/** A view below a rail row, whichever row it belongs to. */
export type RailView = ActivityView | SafetyView | QualityView | AuditView;

/**
 * The tables on this page share the address for their page, search, filters,
 * sort and open row. Moving to another table clears them, so what narrowed
 * one table is never sent as a filter the next one does not know.
 */
export const CLEARED_TABLE_STATE = {
  pageIndex: null,
  query: null,
  fieldFilters: null,
  filterGroups: null,
  sort: null,
  panelType: null,
  panelEntityId: null,
  entityId: null,
  modalType: null,
} as const;
