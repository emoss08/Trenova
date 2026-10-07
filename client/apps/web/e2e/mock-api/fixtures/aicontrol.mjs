// AI control demo data, ported from design_handoff_ai_control/design/data.jsx.
// Times are relative to the scenario anchor; tool names come from the real
// registry (fixtures/ai/tool-catalog.json, dumped from the Go service).
import { readFileSync } from "node:fs";

const load = (name) => JSON.parse(readFileSync(new URL(`./ai/${name}`, import.meta.url), "utf8"));

export const PROVIDER_CATALOG = load("provider-catalog.json");
export const EXTENSION_CATALOG = load("extension-catalog.json");
export const TOOL_CATALOG = load("tool-catalog.json");
export const EVENT_KINDS = load("event-kinds.json");

const HOUR = 3600;
const DAY = 86400;
const TENANT = { organizationId: "org_mock", businessUnitId: "bu_mock" };

const seeded = (seed) => {
  let x = seed;
  return () => (x = (x * 9301 + 49297) % 233280) / 233280;
};

/** Tools whose resource matches one of the prefixes, reads first, capped at `count`. */
function toolsFor(prefixes, count) {
  const matches = TOOL_CATALOG.tools.filter(
    (tool) => !tool.core && prefixes.some((prefix) => tool.resource.startsWith(prefix)),
  );
  const reads = matches.filter((tool) => tool.kind === "query");
  const changes = matches.filter((tool) => tool.kind === "action");
  return [...reads, ...changes].slice(0, count).map((tool) => tool.name);
}

/** Destructive tools only ever propose; small, reversible ones run on their own. */
function risk(name) {
  if (/delete|cancel|void|write_off|transfer|unassign/.test(name)) return 2;
  if (/update|assign|tender|rerate|place|release|split/.test(name)) return 1;
  return 0;
}

function tiersFor(toolNames, [, propose, act]) {
  const actions = toolNames
    .filter((name) => TOOL_CATALOG.tools.find((tool) => tool.name === name)?.kind === "action")
    .sort((a, b) => risk(a) - risk(b));
  const tiers = {};
  actions.forEach((name, index) => {
    if (index < act) tiers[name] = "AutoExecute";
    else if (index < act + propose) tiers[name] = "ActWithApproval";
    else tiers[name] = "Propose";
  });
  return tiers;
}

const AGENT_SPECS = [
  { key: "billing", name: "Billing exceptions", trig: "Chat", icon: "receipt", accent: "amber", template: "BillingAssistant", d: "Works blocked billing queue items and proposes what to do about them.", res: ["billing_queue", "invoice", "accessorial"], tools: 56, tiers: [38, 16, 2], pend: 3, open: 1, last: 12 * 60, runs: [3, 9] },
  { key: "compliance", name: "Compliance desk", trig: "Chat", icon: "shield", accent: "emerald", template: "ComplianceAssistant", d: "Answers questions about driver qualification and what is expiring.", res: ["worker_credential", "qualification", "worker_dot", "dot_random", "worker"], tools: 55, tiers: [49, 6, 0], last: HOUR, runs: [7, 4] },
  { key: "dispatch", name: "Dispatch desk", trig: "Chat", icon: "truck", accent: "indigo", template: "DispatchAssistant", d: "Looks up shipments and drivers for the dispatch team.", res: ["shipment", "tractor", "trailer", "tender"], tools: 56, tiers: [40, 14, 2], pend: 4, open: 4, last: 3 * 60, runs: [11, 14] },
  { key: "fuel", name: "Fuel and IFTA clerk", trig: "Chat", icon: "gauge", accent: "amber", d: "Keeps the fuel tax record: fuel purchases, card statements, state miles, and the quarter's IFTA return for a person to file.", res: ["fuel", "ifta"], tools: 33, tiers: [21, 12, 0], last: DAY, runs: [5, 2] },
  { key: "help", name: "Help", trig: "Chat", icon: "compass", accent: "slate", d: "Explains how to do things in Trenova. Cannot read or change records.", res: [], tools: 0, tiers: [0, 0, 0], last: 8 * 60, runs: [13, 6] },
  { key: "mds", name: "Master data steward", trig: "Chat", icon: "clipboard", accent: "slate", d: "Keeps carriers, customers, commodities, hazmat, locations and equipment right; files scanned paperwork and clears the attention feed.", res: ["carrier", "customer", "commodity", "hazardous", "location", "equipment"], tools: 55, tiers: [33, 22, 0], last: null, runs: [1, 0] },
  { key: "recv", name: "Receivables", trig: "Chat", icon: "banknote", accent: "teal", d: "Works what customers owe once an invoice is out: payments, credit, disputes, late charges, and who to chase first.", res: ["accounts_receivable", "customer_payment", "bank_receipt", "invoice_dispute"], tools: 27, tiers: [17, 10, 0], last: 2 * HOUR, runs: [17, 3] },
  { key: "report", name: "Report analyst", trig: "Chat", icon: "search", accent: "violet", d: "Builds, runs and explains reports and dashboards, and proposes scheduled emails for a person to approve.", res: ["report", "dashboard"], tools: 23, tiers: [19, 4, 0], last: DAY, runs: [19, 2] },
  { key: "settle", name: "Settlements clerk", trig: "Chat", icon: "wallet", accent: "emerald", d: "Runs driver and carrier settlements: drafts the period's pay, matches carrier invoices, and puts every payment in front of a person.", res: ["driver_settlement", "carrier_settlement", "carrier_invoice", "pay_"], tools: 56, tiers: [36, 20, 0], last: 2 * DAY, runs: [23, 2] },
  { key: "workforce", name: "Workforce coordinator", trig: "Chat", icon: "headset", accent: "violet", d: "Handles time off, leave, injuries, reviews, safety records, random testing and permits, and proposes each change.", res: ["worker_", "permit", "performance"], tools: 55, tiers: [30, 25, 0], last: null, runs: [2, 0], roles: ["Dispatch manager", "Safety", "HR"] },
  { key: "cs", name: "Customer service", trig: "Chat", icon: "headset", accent: "sky", template: "CustomerAssistant", d: "Shipment status for the customer-facing team. Off until reviewed.", res: ["shipment", "customer"], tools: 53, tiers: [50, 3, 0], off: true, last: null, runs: [4, 0] },
  { key: "digest", name: "Morning operations digest", trig: "Scheduled", icon: "bell", accent: "amber", d: "A weekday summary of what is stuck, late or unassigned, ready before the desk opens.", res: ["shipment"], tools: 5, tiers: [5, 0, 0], cron: "0 6 * * 1-5", tz: "America/Los_Angeles", last: 5.5 * HOUR, runs: null },
  { key: "billev", name: "Billing exception agent", trig: "Event", icon: "receipt", accent: "amber", template: "BillingException", d: "Diagnoses blocked billing items as they appear and proposes how to clear them.", res: ["billing_queue"], tools: 14, tiers: [10, 4, 0], shadow: true, system: "billing_exception", events: ["billing_queue.item_on_hold", "billing_queue.item_exception"], last: 40 * 60, runs: [29, 5] },
  { key: "coverage", name: "Dispatch coverage agent", trig: "Event", icon: "route", accent: "indigo", template: "DispatchAssignment", d: "Reviews uncovered moves and proposes driver assignments.", res: ["shipment_move", "shipment"], tools: 20, tiers: [14, 6, 0], shadow: true, system: "dispatch_assignment", events: ["shipment_move.unassigned", "shipment_move.coverage_at_risk"], last: 6 * 60, runs: [31, 8] },
  { key: "inbox", name: "Inbox desk", trig: "Event", icon: "inbox", accent: "sky", d: "Works the inbox: files what arrives against the right load, attaches paperwork, answers status questions and puts tenders in front of a person.", res: ["inbound", "document", "shipment"], tools: 25, tiers: [15, 10, 0], shadow: true, events: ["inbound_message.classified", "edi.tender_received"], last: 60, runs: [37, 11] },
];

export const SHADOW_RECORDED = { billev: 18, coverage: 31, inbox: 12 };

function agentId(key) {
  return `agd_${key}`;
}

function buildAgents(anchor, { paused }) {
  return AGENT_SPECS.map((spec, index) => {
    const toolNames = toolsFor(spec.res, spec.tools);
    const knownEvents = new Set(EVENT_KINDS.events.map((event) => event.kind));
    return {
      key: spec.key,
      runsPerDay: spec.runs
        ? Array.from({ length: 14 }, (_, day) => {
            const random = seeded(spec.runs[0]);
            for (let skip = 0; skip < day; skip += 1) random();
            return Math.round(random() * spec.runs[1] * (0.4 + day / 20));
          })
        : [1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1, 0, 1],
      node: {
        id: agentId(spec.key),
        ...TENANT,
        name: spec.name,
        description: spec.d,
        template: spec.template ?? null,
        icon: spec.icon,
        accent: spec.accent,
        instructions: `You are ${spec.name}. ${spec.d}\n\nAnswer from the records you can read and say which record each fact comes from. When a change is needed, propose it and explain why.`,
        guardrails: ["Never change a record a person is editing.", "Never send anything outside the company without approval."],
        toolNames,
        toolTiers: tiersFor(toolNames, spec.tiers),
        autonomyCeiling: spec.tiers[2] > 0 ? "AutoExecute" : "ActWithApproval",
        dataAccessCeiling: "Internal",
        enabled: !spec.off,
        shadowMode: !!spec.shadow || paused,
        decisionTimeoutSeconds: DAY,
        triggerMode: spec.trig,
        cronExpression: spec.cron ?? "",
        cronTimezone: spec.tz ?? "",
        eventKinds: (spec.events ?? []).filter((kind) => knownEvents.has(kind)),
        intervalSeconds: 0,
        endsAt: null,
        maxConcurrentRuns: 1,
        runTimeoutSeconds: 600,
        maxToolCalls: 40,
        monthlyBudgetUsd: index % 3 === 0 ? "50" : null,
        dailyRunLimit: 0,
        toolDailyLimits: {},
        simulationMode: false,
        memoryTokenBudget: null,
        learningOff: false,
        contextProviders: [],
        outputMode: spec.trig === "Scheduled" ? "Report" : "Conversational",
        preferredProviderId: null,
        systemKey: spec.system ?? "",
        starters: [],
        delegateIds: [],
        delegates: [],
        accessMode: spec.roles ? "Roles" : "Everyone",
        accessRoles: (spec.roles ?? []).map((name, roleIndex) => ({ id: `rol_${roleIndex}`, name })),
        lastRunAt: spec.last == null ? null : anchor - spec.last,
        nextRunAt: spec.cron ? anchor - (anchor % DAY) + DAY + 13 * HOUR : null,
        pendingProposals: spec.pend ?? 0,
        openRuns: spec.open ?? 0,
        version: 3,
        createdAt: anchor - 90 * DAY,
        updatedAt: anchor - 2 * DAY,
      },
    };
  });
}

const PROVIDER_SPECS = {
  configured: [
    { key: "anthropic", name: "Anthropic", kind: "AnthropicMessages", model: "claude-sonnet-5", on: false, key2: false, trusted: true, tasks: ["BillingDiagnosis", "AssistantChat"], priority: 10, price: [3, 15] },
    { key: "ollama", name: "Local Ollama", kind: "Ollama", model: "qwen2.5:14b", on: true, trusted: false, priv: true, base: "http://localhost:11434", tasks: ["ScopeClassification", "DocumentClassification", "OperationalInsights", "DailyBriefing", "General"], priority: 20, test: [true, "Connected", 386, 6 * HOUR], week: { calls: 672, failed: 5, ms: 410, tokens: 900000 } },
    { key: "vllm", name: "Workstation vLLM", kind: "OpenAIChat", model: "meta-llama/Llama-3.3-70B-Instruct", on: true, trusted: false, priv: true, base: "http://localhost:8000/v1", tasks: ["AssistantChat", "DocumentExtraction", "FormulaAssistant", "OperationalInsights", "General"], priority: 30, test: [false, "Could not connect", 0, 6.2 * HOUR], week: { calls: 612, failed: 24, ms: 1420, tokens: 2200000 }, failing: 24 },
  ],
  many: [
    { key: "openai", name: "OpenAI", kind: "OpenAIResponses", model: "gpt-5.1-mini", on: true, key2: true, trusted: true, tasks: ["DocumentClassification", "ScopeClassification"], priority: 10, price: [0.4, 1.6], test: [true, "Connected", 290, 3.5 * HOUR], week: { calls: 1840, failed: 2, ms: 310, tokens: 1400000 } },
    { key: "groq", name: "Groq", kind: "OpenAIChat", model: "llama-3.3-70b-versatile", on: true, key2: true, base: "https://api.groq.com/openai/v1", tasks: ["OperationalInsights", "General"], priority: 20, price: [0.59, 0.79], test: [true, "Connected", 120, 4 * HOUR], week: { calls: 960, failed: 0, ms: 140, tokens: 800000 } },
    { key: "azure", name: "Azure OpenAI", kind: "OpenAIChat", model: "gpt-5.1", on: true, key2: true, trusted: true, base: "https://trenova.openai.azure.com/openai/v1", tasks: ["DocumentExtraction", "BillingDiagnosis", "AssistantChat"], priority: 30, price: [1.25, 10], test: [true, "Connected", 520, DAY], week: { calls: 412, failed: 1, ms: 980, tokens: 1100000 } },
    { key: "gemini", name: "Google Gemini", kind: "OpenAIChat", model: "gemini-3-flash", on: false, key2: true, base: "https://generativelanguage.googleapis.com/v1beta/openai", tasks: ["DailyBriefing"], priority: 40, test: [true, "Connected", 340, 4 * DAY] },
    { key: "mistral", name: "Mistral", kind: "OpenAIChat", model: "mistral-large-3", on: false, base: "https://api.mistral.ai/v1", tasks: [], priority: 50 },
  ],
  none: [],
};

function buildProviders(anchor, mode) {
  return (PROVIDER_SPECS[mode] ?? []).map((spec) => ({
    key: spec.key,
    week: spec.week ?? null,
    failing: spec.failing ?? 0,
    node: {
      id: `aip_${spec.key}`,
      ...TENANT,
      name: spec.name,
      description: "",
      kind: spec.kind,
      baseUrl: spec.base ?? "",
      model: spec.model,
      hasApiKey: !!spec.key2,
      allowPrivateNetwork: !!spec.priv,
      structuredOutputMode: spec.kind === "OpenAIChat" ? "JSONMode" : "JSONSchema",
      reasoningEffort: "Off",
      thinkingStyle: "Auto",
      extraBody: null,
      inputCostPerMillion: spec.price ? String(spec.price[0]) : null,
      outputCostPerMillion: spec.price ? String(spec.price[1]) : null,
      maxTokens: 8192,
      tasks: spec.tasks,
      priority: spec.priority,
      embeddingDimensions: null,
      embeddingInputStyle: "None",
      trusted: !!spec.trusted,
      enabled: spec.on,
      lastTest: spec.test
        ? {
            success: spec.test[0],
            message: spec.test[1],
            modelIdentifier: spec.test[0] ? spec.model : "",
            schemaHonoured: spec.test[0],
            latencyMs: spec.test[2],
            detail: spec.test[0]
              ? ""
              : `execute provider request: Post "${spec.base}/chat/completions": dial tcp 127.0.0.1:8000: connect: connection refused`,
            testedAt: anchor - spec.test[3],
          }
        : null,
      version: 4,
      createdAt: anchor - 60 * DAY,
      updatedAt: anchor - 3 * DAY,
    },
  }));
}

const FEATURES = [
  ["AgentTurn", 612, 24, 1700000, 500000, 1420, 4100],
  ["TableQuery", 273, 5, 160000, 40000, 210, 640],
  ["DocumentIntelligenceRoute", 204, 0, 240000, 60000, 380, 910],
  ["DocumentIntelligenceExtract", 188, 0, 310000, 90000, 920, 2300],
  ["FormulaGenerate", 7, 0, 30000, 10000, 2610, 3900],
];

const WEEK = [
  [168, 0],
  [191, 1],
  [74, 0],
  [61, 0],
  [244, 2],
  [268, 18],
  [249, 8],
];

function buildUsage(anchor, providers) {
  if (providers.length === 0) return { days: [], features: [] };
  const priced = providers.some((provider) => provider.node.enabled && provider.node.inputCostPerMillion);
  const today = anchor - (anchor % DAY);
  const days = WEEK.map(([calls, failed], index) => ({
    day: new Date((today - (WEEK.length - 1 - index) * DAY) * 1000).toISOString().slice(0, 10),
    calls,
    failed,
    inputTokens: calls * 3100,
    outputTokens: calls * 900,
    costUsd: priced ? String(Math.round(calls * 0.0042 * 100) / 100) : "0",
    pricedCalls: priced ? calls : 0,
    latencyP50Ms: 640 + index * 40,
    latencyP95Ms: 2900 + index * 120,
  }));
  const features = FEATURES.map(([feature, calls, failed, input, output, p50, p95]) => ({
    feature,
    calls,
    failed,
    inputTokens: input,
    outputTokens: output,
    reasoningTokens: 0,
    costUsd: priced ? String(Math.round(calls * 0.0042 * 100) / 100) : "0",
    pricedCalls: priced ? calls : 0,
    latencyP50Ms: p50,
    latencyP95Ms: p95,
  }));
  return { days, features };
}

const WORKING = [
  ["dispatch", "Diagnosing", "Checking who can cover SHP-48302 before 14:00"],
  ["inbox", "GatheringContext", "Reading a rate confirmation from Ridgeline Freight"],
  ["coverage", "Diagnosing", "Matching drivers to 3 uncovered moves"],
];

function buildRuns(anchor, agents) {
  const runs = [];
  WORKING.forEach(([key, status, summary], index) => {
    const agent = agents.find((candidate) => candidate.key === key);
    if (!agent || !agent.node.enabled) return;
    runs.push(runNode(anchor, agent, { id: `run_live_${index}`, status, summary, startedAt: anchor - 40 * (index + 1) }));
  });
  agents.forEach((agent, agentIndex) => {
    for (let index = 0; index < 4; index += 1) {
      const failed = (agentIndex + index) % 9 === 0;
      runs.push(
        runNode(anchor, agent, {
          id: `run_${agent.key}_${index}`,
          status: failed ? "Failed" : agent.node.shadowMode ? "ShadowCompleted" : "Completed",
          summary: failed ? "The model did not answer in time" : `Answered a question about ${agent.node.name.toLowerCase()}`,
          startedAt: anchor - (index + 1) * 3 * HOUR - agentIndex * 400,
          completedAt: anchor - (index + 1) * 3 * HOUR - agentIndex * 400 + 42,
          errorMessage: failed ? "provider timed out after 60s" : "",
        }),
      );
    }
  });
  return runs;
}

function runNode(anchor, agent, values) {
  return {
    ...TENANT,
    agentType: agent.node.template === "BillingException" ? "BillingException" : agent.node.template === "DispatchAssignment" ? "DispatchAssignment" : "General",
    agentDefinitionId: agent.node.id,
    trigger: agent.node.triggerMode === "Chat" ? "Chat" : agent.node.triggerMode,
    subjectType: "Organization",
    subjectId: "org_mock",
    workflowId: `wf_${values.id}`,
    modelIdentifier: "claude-sonnet-5",
    promptVersion: "v3",
    completedAt: null,
    errorMessage: "",
    traceId: "",
    traceUrl: null,
    parentOwnerKind: null,
    delegateCallId: "",
    handedBy: null,
    version: 1,
    createdAt: values.startedAt,
    updatedAt: values.completedAt ?? values.startedAt,
    ...values,
  };
}

export function createAIState(anchor, scenario) {
  const providers = buildProviders(anchor, scenario.aiProviders);
  const paused = !!scenario.aiPaused;
  const agents = buildAgents(anchor, { paused });
  return {
    providers,
    agents,
    runs: buildRuns(anchor, agents),
    usage: buildUsage(anchor, providers),
    dismissedFailures: new Map(),
    tuneUps: buildTuneUps(anchor, agents, providers),
    control: {
      id: "agc_mock",
      ...TENANT,
      shadowMode: paused,
      earnedAutonomy: false,
      promotionThreshold: 10,
      personMonthlyMessages: 250,
      learningOff: false,
      aiTrainingConsent: false,
      aiTrainingConsentChangedAt: null,
      version: 7,
      createdAt: anchor - 120 * DAY,
      updatedAt: anchor - 5 * DAY,
    },
  };
}

const EMPTY_EVIDENCE = {
  fromTier: null,
  toTier: null,
  streak: 0,
  approvals: 0,
  approvalsPerWeek: 0,
  rejections: 0,
  calls: 0,
  failed: 0,
  rescued: 0,
  recorded: 0,
  matchRate: 0,
  wouldFail: 0,
  tasks: [],
  model: "",
  lastRunAt: null,
  idleSince: 0,
  tools: 0,
};

/** The prototype's five tune-ups, each kept only while what it names is in the scenario. */
function buildTuneUps(anchor, agents, providers) {
  const agent = (key) => agents.find((candidate) => candidate.key === key);
  const provider = (key) => providers.find((candidate) => candidate.key === key);
  const out = [];
  const add = (id, kind, values, evidence) =>
    out.push({
      id: `aitu_${id}`,
      kind,
      agent: null,
      provider: null,
      otherProvider: null,
      toolName: null,
      task: null,
      status: "Open",
      dismissedUntil: null,
      computedAt: anchor - 8 * HOUR,
      version: 1,
      ...values,
      evidence: { ...EMPTY_EVIDENCE, ...evidence },
    });

  const dispatch = agent("dispatch");
  const raised = Object.entries(dispatch.node.toolTiers).find(
    ([, tier]) => tier === "ActWithApproval",
  );
  if (raised) {
    add("raise", "RaiseToolTier", { agentKey: "dispatch", toolName: raised[0] }, {
      fromTier: raised[1],
      toTier: raised[1] === "Propose" ? "ActWithApproval" : "AutoExecute",
      streak: 9,
      approvals: 88,
      approvalsPerWeek: 20.5,
      rejections: 0,
    });
  }
  if (provider("ollama") && provider("vllm")) {
    add("order", "ReorderProviders", { providerKey: "ollama", otherProviderKey: "vllm" }, {
      calls: 672,
      failed: 24,
      rescued: 24,
      tasks: ["OperationalInsights", "General"],
    });
  }
  add("live", "LeaveShadow", { agentKey: "coverage" }, { recorded: 31, matchRate: 0.87, wouldFail: 0 });
  if (provider("ollama")) {
    add("embed", "AssignTask", { providerKey: "ollama", task: "Embedding" }, { model: "nomic-embed-text" });
  }
  const steward = agent("mds");
  add("idle", "TurnOffIdleAgent", { agentKey: "mds" }, {
    lastRunAt: anchor - 14 * 24 * HOUR,
    idleSince: anchor - 14 * 24 * HOUR,
    tools: steward.node.toolNames.length,
  });
  return out;
}
