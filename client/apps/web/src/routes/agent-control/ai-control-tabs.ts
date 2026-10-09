import { parseAsArrayOf, parseAsBoolean, parseAsString, parseAsStringLiteral } from "nuqs";

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

/** The Agents tab's filter: every agent, those with proposals waiting, those in shadow, or those off. */
export const agentFilters = ["all", "waiting", "shadow", "off"] as const;
export type AgentFilter = (typeof agentFilters)[number];

export const AGENT_FILTER_PARAM = "agents";

export const agentFilterParser = parseAsStringLiteral(agentFilters)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("all");

/** The agent open in the Agents tab's sheet; a link from anywhere on the page opens one. */
export const AGENT_OPEN_PARAM = "openAgent";

export const agentOpenParser = parseAsString.withOptions({ history: "push", shallow: true });

/**
 * The agent builder is a page of its own over the Agents tab, not a table's
 * panel, so it keeps its own keys: the agent it edits, or what a new one
 * starts from.
 */
export const BUILDER_STARTS = ["chat", "scheduled", "event", "blank"] as const;
export type BuilderStart = (typeof BUILDER_STARTS)[number];

export const AGENT_EDIT_PARAM = "editAgent";
export const agentEditParser = parseAsString.withOptions({ history: "push", shallow: true });

export const AGENT_NEW_PARAM = "newAgent";
export const agentNewParser = parseAsStringLiteral(BUILDER_STARTS).withOptions({
  history: "push",
  shallow: true,
});

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
 * extraction.
 */
export const qualityViews = ["agents", "runs", "ratings", "golden", "extraction"] as const;
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

/**
 * Document extraction is six views: accuracy, corrections, the evaluation set,
 * its runs, the candidate shadowing production, and the candidate rolled out to
 * a share of it.
 */
export const extractionViews = [
  "accuracy",
  "corrections",
  "cases",
  "runs",
  "shadow",
  "rollout",
] as const;
export type ExtractionView = (typeof extractionViews)[number];

export const EXTRACTION_VIEW_PARAM = "extraction";

export const extractionViewParser = parseAsStringLiteral(extractionViews)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("accuracy");

/**
 * The page's dialogs and sheets that are not a table's own panel, each opened
 * by a key in the address so a link, a reload or the back button lands on it.
 * A table's panel keeps the shared panel keys; these sit beside them.
 */
const dialogOptions = { history: "push", shallow: true } as const;

/** The overview's organization-wide settings editor. */
export const POLICY_EDITOR_PARAM = "policy";
export const policyEditorParser = parseAsBoolean.withOptions(dialogOptions);

/** The provider open in its editor; the provider's read sheet is the panel keys. */
export const PROVIDER_EDITOR_PARAM = "editProvider";
export const providerEditorParser = parseAsString.withOptions(dialogOptions);

/** The extension open in its sheet, by type. */
export const EXTENSION_OPEN_PARAM = "extension";
export const extensionOpenParser = parseAsString.withOptions(dialogOptions);

/** The agent open in Quality's sheet from the figures above the tables. */
export const QUALITY_SHEET_PARAM = "qualitySheet";
export const qualitySheetParser = parseAsString.withOptions(dialogOptions);

/** Quality's sweep settings editor. */
export const SWEEP_SETTINGS_PARAM = "sweepSettings";
export const sweepSettingsParser = parseAsBoolean.withOptions(dialogOptions);

/** The dialog that exports the audit trail, from either of its views. */
export const AUDIT_EXPORT_PARAM = "exportTrail";
export const auditExportParser = parseAsBoolean.withOptions(dialogOptions);

/** Document extraction's dialogs: a new evaluation run, the shadow's and the rollout's settings. */
export const extractionDialogs = ["newRun", "shadow", "rollout"] as const;
export type ExtractionDialog = (typeof extractionDialogs)[number];

export const EXTRACTION_DIALOG_PARAM = "extractionDialog";
export const extractionDialogParser =
  parseAsStringLiteral(extractionDialogs).withOptions(dialogOptions);

/** The retrieval source asked to be indexed again. */
export const REINDEX_PARAM = "reindex";
export const reindexParser = parseAsStringLiteral(retrievalSourceValues).withOptions(dialogOptions);

/** The memory kind a first memory starts from, when there are none to list yet. */
export const memoryStarterKinds = ["Instruction", "Fact", "Procedure"] as const;
export type MemoryStarterKind = (typeof memoryStarterKinds)[number];

export const MEMORY_STARTER_PARAM = "newMemory";
export const memoryStarterParser =
  parseAsStringLiteral(memoryStarterKinds).withOptions(dialogOptions);

/** The agent's memory suggestion open to be reworded before it is approved. */
export const MEMORY_SUGGESTION_PARAM = "suggestion";
export const memorySuggestionParser = parseAsString.withOptions(dialogOptions);

/** The evaluation open in its detail dialog. */
export const EVALUATION_OPEN_PARAM = "evaluation";
export const evaluationOpenParser = parseAsString.withOptions(dialogOptions);

/** The run opened over a proposal's or an exception's sheet. */
export const RUN_OPEN_PARAM = "run";
export const runOpenParser = parseAsString.withOptions(dialogOptions);

/** The rule editor over a tool's sheet. */
export const TOOL_RULE_EDIT_PARAM = "editRule";
export const toolRuleEditParser = parseAsBoolean.withOptions(dialogOptions);

/** The agent builder's try panel. */
export const AGENT_TRY_PARAM = "tryAgent";
export const agentTryParser = parseAsBoolean.withOptions(dialogOptions);

/** The agent builder's tool picker. */
export const TOOL_PICKER_PARAM = "pickTools";
export const toolPickerParser = parseAsBoolean.withOptions(dialogOptions);

export const aiControlDialogParsers = {
  [POLICY_EDITOR_PARAM]: policyEditorParser,
  [PROVIDER_EDITOR_PARAM]: providerEditorParser,
  [EXTENSION_OPEN_PARAM]: extensionOpenParser,
  [QUALITY_SHEET_PARAM]: qualitySheetParser,
  [SWEEP_SETTINGS_PARAM]: sweepSettingsParser,
  [AUDIT_EXPORT_PARAM]: auditExportParser,
  [EXTRACTION_DIALOG_PARAM]: extractionDialogParser,
  [REINDEX_PARAM]: reindexParser,
  [MEMORY_STARTER_PARAM]: memoryStarterParser,
  [MEMORY_SUGGESTION_PARAM]: memorySuggestionParser,
  [EVALUATION_OPEN_PARAM]: evaluationOpenParser,
  [RUN_OPEN_PARAM]: runOpenParser,
  [TOOL_RULE_EDIT_PARAM]: toolRuleEditParser,
  [AGENT_TRY_PARAM]: agentTryParser,
  [TOOL_PICKER_PARAM]: toolPickerParser,
};

/** Every dialog closed: a move to another table never carries one open into it. */
export const CLEARED_DIALOG_STATE = {
  [POLICY_EDITOR_PARAM]: null,
  [PROVIDER_EDITOR_PARAM]: null,
  [EXTENSION_OPEN_PARAM]: null,
  [QUALITY_SHEET_PARAM]: null,
  [SWEEP_SETTINGS_PARAM]: null,
  [AUDIT_EXPORT_PARAM]: null,
  [EXTRACTION_DIALOG_PARAM]: null,
  [REINDEX_PARAM]: null,
  [MEMORY_STARTER_PARAM]: null,
  [MEMORY_SUGGESTION_PARAM]: null,
  [EVALUATION_OPEN_PARAM]: null,
  [RUN_OPEN_PARAM]: null,
  [TOOL_RULE_EDIT_PARAM]: null,
  [AGENT_TRY_PARAM]: null,
  [TOOL_PICKER_PARAM]: null,
} as const satisfies Record<keyof typeof aiControlDialogParsers, null>;
