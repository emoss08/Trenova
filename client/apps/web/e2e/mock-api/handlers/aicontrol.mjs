// AI control: the GraphQL operations and REST routes /admin/agent-control sends.
import {
  EVENT_KINDS,
  EXTENSION_CATALOG,
  PROVIDER_CATALOG,
  ROLES,
  SHADOW_RECORDED,
  TOOL_CATALOG,
} from "../fixtures/aicontrol.mjs";

const DAY = 86400;
const WORKING_STATUSES = new Set(["Pending", "GatheringContext", "Diagnosing"]);
const ALL_TASKS = PROVIDER_CATALOG.tasks.map((task) => task.task);
const TRUSTED_TASKS = new Set(["BillingDiagnosis"]);

function compare(left, right) {
  if (left === right) return 0;
  if (left == null) return 1;
  if (right == null) return -1;
  return left < right ? -1 : 1;
}

function matches(value, { operator, value: expected }) {
  switch (operator) {
    case "eq":
      return value === expected;
    case "ne":
      return value !== expected;
    case "in":
      return Array.isArray(expected) && expected.includes(value);
    case "notin":
      return Array.isArray(expected) && !expected.includes(value);
    case "gt":
      return value > expected;
    case "gte":
      return value >= expected;
    case "lt":
      return value < expected;
    case "lte":
      return value <= expected;
    case "contains":
      return String(value ?? "").toLowerCase().includes(String(expected).toLowerCase());
    case "isnull":
      return value == null;
    case "isnotnull":
      return value != null;
    default:
      return true;
  }
}

/** A cursor connection over in-memory rows, honouring the data table's input. */
export function connection(rows, input = {}, { searchOf, includeTotalCount = true } = {}) {
  let filtered = rows;
  const query = (input.query ?? "").trim().toLowerCase();
  if (query && searchOf) {
    filtered = filtered.filter((row) => searchOf(row).toLowerCase().includes(query));
  }
  for (const filter of input.fieldFilters ?? []) {
    filtered = filtered.filter((row) => matches(row[filter.field], filter));
  }
  for (const group of input.filterGroups ?? []) {
    filtered = filtered.filter((row) => (group.filters ?? []).some((filter) => matches(row[filter.field], filter)));
  }
  const sorted = [...filtered];
  for (const sort of [...(input.sort ?? [])].reverse()) {
    sorted.sort((a, b) => compare(a[sort.field], b[sort.field]) * (sort.direction === "desc" ? -1 : 1));
  }
  const offset = input.after ? Number(Buffer.from(input.after, "base64").toString().split(":")[1]) + 1 : 0;
  const first = input.first ?? 20;
  const page = sorted.slice(offset, offset + first);
  const cursorAt = (index) => Buffer.from(`mock:${offset + index}`).toString("base64");
  return {
    edges: page.map((node, index) => ({ cursor: cursorAt(index), node })),
    ...(includeTotalCount ? { totalCount: sorted.length } : {}),
    pageInfo: {
      hasNextPage: offset + first < sorted.length,
      endCursor: page.length ? cursorAt(page.length - 1) : null,
      hasPreviousPage: offset > 0,
      startCursor: page.length ? cursorAt(0) : null,
    },
  };
}

function enabledProviders(ai) {
  return ai.providers.filter((provider) => provider.node.enabled).sort((a, b) => a.node.priority - b.node.priority);
}

function canServe(provider, task) {
  return provider.node.tasks.includes(task) && (!TRUSTED_TASKS.has(task) || provider.node.trusted);
}

export function uncoveredTasks(ai) {
  const enabled = enabledProviders(ai);
  return ALL_TASKS.filter((task) => !enabled.some((provider) => canServe(provider, task)));
}

function failing(ai) {
  return enabledProviders(ai)
    .filter((provider) => provider.failing > 0)
    .map((provider) => ({
      providerId: provider.node.id,
      name: provider.node.name,
      failedCalls: provider.failing,
      lastFailureAt: provider.node.lastTest?.testedAt ?? 0,
    }));
}

function facts(state) {
  const ai = state.ai;
  const on = ai.agents.filter((agent) => agent.node.enabled);
  const shadow = on.filter((agent) => agent.node.shadowMode);
  const keyless = ai.providers.find((provider) => !provider.node.hasApiKey && provider.node.kind !== "Ollama" && !provider.node.allowPrivateNetwork);
  return {
    providersOn: enabledProviders(ai).length,
    providersTotal: ai.providers.length,
    weekCalls: ai.usage.days.reduce((sum, day) => sum + day.calls, 0),
    awaitingKey: keyless ? { providerId: keyless.node.id, name: keyless.node.name } : null,
    uncovered: uncoveredTasks(ai).length,
    paused: ai.control.shadowMode,
    agents: {
      total: ai.agents.length,
      on: on.length,
      working: ai.runs.filter((run) => WORKING_STATUSES.has(run.status)).length,
      waiting: on.reduce((sum, agent) => sum + agent.node.pendingProposals, 0),
      shadow: shadow.length,
      shadowRecorded: shadow.reduce((sum, agent) => sum + (SHADOW_RECORDED[agent.key] ?? 0), 0),
    },
    failing: failing(ai),
  };
}

const seg = (text, extra = {}) => ({ text, strong: false, target: null, providerId: null, tone: null, ...extra });
const strong = (text) => seg(text, { strong: true });
const link = (text, target, tone = null, providerId = null) => seg(text, { target, tone, providerId });
const count = (n, one, many) => (n === 1 ? `1 ${one}` : `${n} ${many}`);
const isAre = (n) => (n === 1 ? "is" : "are");

const grouped = (n) => n.toLocaleString("en-US");
const plural = (n, one, many) => (n === 1 ? `1 ${one}` : `${grouped(n)} ${many}`);

function failingLink(f) {
  if (f.failing.length === 0) return [];
  if (f.failing.length === 1) return [link(f.failing[0].name, "provider", "danger", f.failing[0].providerId)];
  return [link(count(f.failing.length, "provider", "providers"), "providers", "danger")];
}

function uncoveredClause(f, tone) {
  if (!f.uncovered) return [];
  return [link(`${count(f.uncovered, "task", "tasks")} ${f.uncovered === 1 ? "has" : "have"}`, "routing", tone), seg(" nowhere to go. ")];
}

function trimEnd(segments) {
  if (segments.length) segments[segments.length - 1].text = segments[segments.length - 1].text.trimEnd();
  return segments;
}

/** Mirrors aicontrolsummary.Plain in the Go service. */
function sentence(tab, f) {
  const waits = (n) => (n === 1 ? "waits" : "wait");
  if (tab === "Agents") {
    const out = [strong(`${f.agents.on} of ${f.agents.total}`), seg(` agents ${isAre(f.agents.on)} on`)];
    if (f.agents.working) out.push(seg(", and "), strong(String(f.agents.working)), seg(` ${isAre(f.agents.working)} working right now`));
    out.push(seg(". "));
    if (f.agents.waiting) out.push(link(count(f.agents.waiting, "proposal", "proposals"), "agents:waiting", "warn"), seg(` ${waits(f.agents.waiting)} on a person. `));
    if (f.agents.shadow) {
      const one = f.agents.shadow === 1;
      out.push(seg(`${count(f.agents.shadow, "agent", "agents")} ${one ? "runs" : "run"} in `), link("shadow", "agents:shadow"), seg(` and ${one ? "has" : "have"} recorded ${plural(f.agents.shadowRecorded, "proposal", "proposals")} nobody has seen — worth a look before you let ${one ? "it" : "them"} go live. `));
    }
    return trimEnd(out);
  }
  if (tab === "Providers") {
    if (!f.providersOn) return [seg("No provider is on. "), seg("Connect one and every AI feature starts working.")];
    const out = [strong(`${f.providersOn} of ${f.providersTotal}`), seg(` providers ${isAre(f.providersOn)} taking work — ${plural(f.weekCalls, "call", "calls")} this week. `)];
    if (f.failing.length === 1) out.push(...failingLink(f), seg(" is failing to connect, so its tasks fall through to the next in line. "));
    else if (f.failing.length > 1) out.push(...failingLink(f), seg(" are failing to connect, so their tasks fall through to the next in line. "));
    if (f.awaitingKey) out.push(link(f.awaitingKey.name, "provider", "warn", f.awaitingKey.providerId), seg(" is waiting for a key. "));
    out.push(...uncoveredClause(f, null));
    return trimEnd(out);
  }
  if (!f.providersOn) {
    return [seg("Nothing can answer yet. Your "), strong(count(f.agents.total, "agent", "agents")), seg(` ${isAre(f.agents.total)} set up and waiting for a model provider — connect one and they start on their own.`)];
  }
  if (f.paused) {
    const out = [seg("Every agent is "), seg("paused", { strong: true, tone: "warn" }), seg(". They keep running and recording what they would do, but nothing is offered or executed")];
    if (f.agents.waiting) out.push(seg(" — "), link(count(f.agents.waiting, "proposal", "proposals"), "watchtower"), seg(` ${isAre(f.agents.waiting)} held until you resume`));
    out.push(seg("."));
    return out;
  }
  const out = [strong(count(f.agents.on, "agent", "agents")), seg(` ${isAre(f.agents.on)} on`)];
  if (f.agents.working) out.push(seg(", "), strong(String(f.agents.working)), seg(" working right now"));
  out.push(seg(". "));
  const failing = failingLink(f);
  if (f.agents.waiting && failing.length) out.push(link(count(f.agents.waiting, "proposal", "proposals"), "watchtower"), seg(` ${waits(f.agents.waiting)} on a person in Watchtower, and `), ...failing, seg(" can't connect. "));
  else if (f.agents.waiting) out.push(link(count(f.agents.waiting, "proposal", "proposals"), "watchtower"), seg(` ${waits(f.agents.waiting)} on a person in Watchtower. `));
  else if (failing.length) out.push(...failing, seg(" can't connect. "));
  out.push(...uncoveredClause(f, "warn"));
  return trimEnd(out);
}

function summary(state, tab) {
  const f = facts(state);
  const visible = f.failing.filter((failure) => {
    const dismissedAt = state.ai.dismissedFailures.get(failure.providerId);
    return dismissedAt == null || dismissedAt < failure.lastFailureAt;
  });
  return {
    tab,
    segments: sentence(tab, f),
    narrated: state.scenario.ai,
    pending: false,
    factsHash: "mock",
    generatedAt: state.now(),
    facts: f,
    visibleFailures: visible,
  };
}

function promotionPreview(state, threshold) {
  const out = [];
  for (const agent of state.ai.agents) {
    if (!agent.node.enabled) continue;
    Object.entries(agent.node.toolTiers)
      .filter(([, tier]) => tier === "Propose")
      .slice(0, agent.key === "dispatch" ? 2 : agent.key === "billing" ? 1 : 0)
      .forEach(([toolName], index) => {
        const streak = threshold + 2 - index;
        out.push({ agentDefinitionId: agent.node.id, agentName: agent.node.name, toolName, streak, from: "Propose", to: "ActWithApproval" });
      });
  }
  return out;
}

function featureNodes(state) {
  return state.ai.usage.features.map(({ reasoningTokens: _reasoning, ...feature }) => feature);
}

function usageSummary(state) {
  const features = state.ai.usage.features;
  const sum = (key) => features.reduce((total, feature) => total + feature[key], 0);
  const cost = features.reduce((total, feature) => total + Number(feature.costUsd), 0);
  return {
    since: state.now() - 7 * DAY,
    calls: sum("calls"),
    failed: sum("failed"),
    inputTokens: sum("inputTokens"),
    outputTokens: sum("outputTokens"),
    reasoningTokens: 0,
    costUsd: String(Math.round(cost * 100) / 100),
    pricedCalls: sum("pricedCalls"),
    latencyP50Ms: features.length ? 720 : 0,
    latencyP95Ms: features.length ? 3400 : 0,
    byProvider: state.ai.providers
      .filter((provider) => provider.week)
      .map((provider) => ({
        providerId: provider.node.id,
        providerName: provider.node.name,
        model: provider.node.model,
        calls: provider.week.calls,
        failed: provider.week.failed,
        inputTokens: Math.round(provider.week.tokens * 0.75),
        outputTokens: Math.round(provider.week.tokens * 0.25),
        reasoningTokens: 0,
        costUsd: provider.node.inputCostPerMillion ? String(Math.round(provider.week.calls * 0.004 * 100) / 100) : "0",
        pricedCalls: provider.node.inputCostPerMillion ? provider.week.calls : 0,
        latencyP50Ms: provider.week.ms,
        latencyP95Ms: provider.week.ms * 3,
      })),
    byFeature: features,
    recentFailures: failing(state.ai).map((failure) => ({
      providerId: failure.providerId,
      providerName: failure.name,
      model: state.ai.providers.find((provider) => provider.node.id === failure.providerId)?.node.model ?? "",
      task: "AssistantChat",
      errorClass: "Connection",
      message: "Could not connect",
      at: failure.lastFailureAt,
    })),
  };
}

const agentSearch = (node) => `${node.name} ${node.description}`;

function tuneUpNode(state, tuneUp) {
  const ai = state.ai;
  const agent = tuneUp.agentKey ? ai.agents.find((candidate) => candidate.key === tuneUp.agentKey) : null;
  const providerOf = (key) => {
    const found = key ? ai.providers.find((candidate) => candidate.key === key) : null;
    return found ? { id: found.node.id, name: found.node.name, kind: found.node.kind } : null;
  };
  const { agentKey: _agent, providerKey, otherProviderKey, ...rest } = tuneUp;
  return {
    ...rest,
    agent: agent ? { id: agent.node.id, name: agent.node.name, icon: agent.node.icon, accent: agent.node.accent } : null,
    provider: providerOf(providerKey),
    otherProvider: providerOf(otherProviderKey),
  };
}

function visibleTuneUp(state, tuneUp) {
  if (tuneUp.status === "Open") return true;
  return tuneUp.status === "Dismissed" && tuneUp.dismissedUntil <= state.now();
}

function decideTuneUp(state, id, version, change) {
  const tuneUp = state.ai.tuneUps.find((candidate) => candidate.id === id);
  if (!tuneUp) throw new Error("AITuneUp not found");
  if (tuneUp.version !== version) throw new Error("Someone else saved changes while you were editing");
  change(tuneUp);
  tuneUp.version += 1;
  return tuneUpNode(state, tuneUp);
}

function applyTuneUp(state, tuneUp) {
  const ai = state.ai;
  const agent = ai.agents.find((candidate) => candidate.key === tuneUp.agentKey);
  const provider = ai.providers.find((candidate) => candidate.key === tuneUp.providerKey);
  switch (tuneUp.kind) {
    case "RaiseToolTier":
      agent.node.toolTiers[tuneUp.toolName] = tuneUp.evidence.toTier;
      break;
    case "LeaveShadow":
      agent.node.shadowMode = false;
      break;
    case "TurnOffIdleAgent":
      agent.node.enabled = false;
      break;
    case "AssignTask":
      provider.node.tasks = [...provider.node.tasks, tuneUp.task];
      break;
    case "ReorderProviders": {
      const behind = ai.providers.find((candidate) => candidate.key === tuneUp.otherProviderKey);
      provider.node.priority = behind.node.priority - 1;
      break;
    }
  }
  tuneUp.status = "Applied";
}

export const AI_HANDLERS = {
  AITuneUps: (state) => {
    const visible = state.ai.tuneUps.filter((tuneUp) => visibleTuneUp(state, tuneUp));
    return {
      aiTuneUps: {
        computedAt: state.ai.tuneUps[0]?.computedAt ?? null,
        windowDays: 30,
        items: state.ai.providers.length ? visible.map((tuneUp) => tuneUpNode(state, tuneUp)) : [],
      },
    };
  },
  ApplyAITuneUp: (state, v) => ({
    applyAITuneUp: decideTuneUp(state, v.id, v.version, (tuneUp) => applyTuneUp(state, tuneUp)),
  }),
  DismissAITuneUp: (state, v) => ({
    dismissAITuneUp: decideTuneUp(state, v.id, v.version, (tuneUp) => {
      tuneUp.status = "Dismissed";
      tuneUp.dismissedUntil = state.now() + (v.days ?? 30) * DAY;
    }),
  }),
  RestoreAITuneUp: (state, v) => ({
    restoreAITuneUp: decideTuneUp(state, v.id, v.version, (tuneUp) => {
      tuneUp.status = "Open";
      tuneUp.dismissedUntil = null;
    }),
  }),
  AIControlSummary: (state, v) => ({ aiControlSummary: summary(state, v.tab) }),
  DismissAIProviderFailure: (state, v) => {
    state.ai.dismissedFailures.set(v.providerId, v.lastFailureAt);
    return { dismissAIProviderFailure: true };
  },
  RestoreAIProviderFailure: (state, v) => {
    state.ai.dismissedFailures.delete(v.providerId);
    return { restoreAIProviderFailure: true };
  },
  AgentPromotionPreview: (state, v) => ({ agentPromotionPreview: promotionPreview(state, v.threshold) }),
  AIUsageDaily: (state) => ({ aiUsageDaily: state.ai.usage.days }),
  AIUsageFeatures: (state, v) => ({
    aiUsageFeatures: connection(featureNodes(state), v.input, {
      searchOf: (row) => row.feature,
      includeTotalCount: v.includeTotalCount !== false,
    }),
  }),
  AIUsageSummary: (state) => ({ aiUsageSummary: usageSummary(state) }),
  AgentControlSettings: (state) => ({ agentControl: state.ai.control }),
  AITrainingExportHistory: () => ({ aiTrainingExportHistory: [] }),
  UpdateAgentControl: (state, v) => {
    const { version: _version, ...input } = v.input;
    Object.assign(state.ai.control, input, { version: state.ai.control.version + 1, updatedAt: state.now() });
    for (const agent of state.ai.agents) {
      if (input.shadowMode === true) agent.node.shadowMode = true;
    }
    return { updateAgentControl: state.ai.control };
  },
  AgentDraftingAvailable: (state) => ({
    agentDraftingAvailable: state.ai.providers.some((provider) => provider.node.enabled),
  }),
  DraftAgentFromDescription: (state, v) => ({
    draftAgentFromDescription: {
      name: "Detention desk",
      description: "Starts the detention clock when a truck waits too long and drafts the charge.",
      icon: "gauge",
      accent: "amber",
      instructions:
        "You are the detention desk for {{organization}}.\n\nWhen a truck has waited more than two hours past its appointment, check the arrival time on the shipment and the customer's detention terms.\nDraft a short note to the customer with the start time and the running total, and propose the detention charge.\nIf the arrival time is missing, flag the shipment for review instead of guessing.",
      guardrails: ["Promise a customer a delivery date or a rate"],
      triggerMode: "Event",
      cronExpression: "",
      cronTimezone: "",
      eventKinds: EVENT_KINDS.events.slice(0, 1).map((event) => event.kind),
      intervalSeconds: 0,
      toolNames: TOOL_CATALOG.tools.filter((tool) => !tool.core && tool.resource.startsWith("shipment")).slice(0, 6).map((tool) => tool.name),
      toolTiers: {},
      autonomyCeiling: "ActWithApproval",
      dataAccessCeiling: "Internal",
      outputMode: "Conversational",
      enabled: true,
      shadowMode: true,
      decisionTimeoutSeconds: DAY,
      runTimeoutSeconds: 600,
      maxToolCalls: 12,
      maxConcurrentRuns: 1,
      notes: v.description.length > 400 ? [{ field: "description", value: "", reason: "The description was shortened" }] : [],
    },
  }),
  TightenAgentInstructions: (_state, v) => ({
    tightenAgentInstructions: {
      instructions: v.instructions.replace(/\n{3,}/g, "\n\n").trim(),
      changed: /\n{3,}/.test(v.instructions),
    },
  }),
  AIAgentRoster: (state) => ({
    aiAgentRoster: state.ai.agents.map(rosterStat),
  }),
  AgentToolRuleTable: (state, v) => ({
    agentToolRuleConnection: connection(TOOL_RULES, v.input, {
      searchOf: (rule) => `${rule.name} ${rule.title}`,
      includeTotalCount: v.includeTotalCount !== false,
    }),
  }),
  AgentInstructionLint: (_state, v) => ({ agentInstructionLint: lintOf(v.input) }),
  AgentShadowReport: (state, v) => {
    const agent = agentOf(state, v.agentId);
    const recorded = agent ? (SHADOW_RECORDED[agent.key] ?? 0) : 0;
    return {
      agentShadowReport: {
        days: v.days ?? 30,
        recorded,
        matched: Math.round(recorded * 0.87),
        matchRate: recorded ? 0.87 : null,
        wouldReject: recorded ? 2 : 0,
        wouldFail: 0,
        unanswered: recorded ? 1 : 0,
      },
    };
  },
  AgentDefinitionVersions: (state, v) => {
    const agent = agentOf(state, v.agentId);
    return { agentDefinitionVersions: agent ? versionsOf(state, agent) : [] };
  },
  AgentDefinitionVersionDraft: (state, v) => {
    const agent = agentOf(state, v.agentId);
    return {
      agentDefinitionVersionDraft: agent
        ? { ...agent.node, autonomyCeiling: "ActWithApproval", guardrails: agent.node.guardrails.slice(0, 1) }
        : null,
    };
  },
  AgentDefinitionCard: (state, v) => ({ agentDefinition: agentOf(state, v.id)?.node ?? null }),
  AgentAccessPreview: (state, v) => ({ agentAccessPreview: accessPreview(state, v.input) }),
  AgentScorecard: (state, v) => {
    const agent = agentOf(state, v.input.agentDefinitionId);
    return { agentScorecard: agent ? scorecardOf(state, agent) : null };
  },
  AgentDefinitionCards: (state, v) => ({
    agentDefinitions: connection(
      state.ai.agents.map((agent) => agent.node),
      v.input,
      { searchOf: agentSearch, includeTotalCount: v.includeTotalCount !== false },
    ),
  }),
  AgentChoices: (state, v) => ({
    agentDefinitions: connection(state.ai.agents.map((agent) => agent.node), v.input, { searchOf: agentSearch }),
  }),
  AgentDefinitionCount: (state, v) => ({
    agentDefinitions: { totalCount: connection(state.ai.agents.map((agent) => agent.node), v.input).totalCount },
  }),
  AgentRunTable: (state, v) => ({
    agentRuns: connection(state.ai.runs, v.input, {
      searchOf: (run) => run.summary,
      includeTotalCount: v.includeTotalCount !== false,
    }),
  }),
  AgentRunCount: (state, v) => ({ agentRuns: { totalCount: connection(state.ai.runs, v.input).totalCount } }),
  AgentProposalCount: (state) => ({
    agentProposals: { totalCount: state.ai.agents.reduce((sum, agent) => sum + agent.node.pendingProposals, 0) },
  }),
  AgentMemoryCount: () => ({ agentMemories: { totalCount: 42 } }),
  AIProviderCards: (state, v) => ({
    aiProviders: connection(
      state.ai.providers.map((provider) => provider.node),
      { ...v.input, sort: v.input?.sort?.length ? v.input.sort : [{ field: "priority", direction: "asc" }] },
      { searchOf: (node) => `${node.name} ${node.model}` },
    ),
  }),
  AIRetrievalStatus: (state) => ({
    aiRetrievalStatus: {
      availability: { available: true, reason: "", extensionInstalled: true, extensionVersion: "0.8.0" },
      settings: {
        memoryEnabled: true,
        documentsEnabled: true,
        inboundMessagesEnabled: false,
        monthlyIndexingBudgetUsd: "25",
        paused: false,
        pausedReason: "",
        pausedAt: null,
        activeModelKey: "openai:text-embedding-3-small:1536",
        dimensions: 1536,
        pendingModelKey: "",
        pendingDimensions: null,
        version: 2,
        updatedAt: state.now() - 9 * DAY,
      },
      sources: [
        { sourceType: "Memory", enabled: true, total: 42, indexed: 42, pending: 0, failed: 0, skipped: 0, lastIndexedAt: state.now() - 3600, lastAttemptAt: state.now() - 3600 },
        { sourceType: "Document", enabled: true, total: 1840, indexed: 1822, pending: 12, failed: 6, skipped: 0, lastIndexedAt: state.now() - 900, lastAttemptAt: state.now() - 900 },
        { sourceType: "InboundMessage", enabled: false, total: 0, indexed: 0, pending: 0, failed: 0, skipped: 0, lastIndexedAt: null, lastAttemptAt: null },
      ],
      monthStartedAt: state.now() - 6 * DAY,
      indexingCostMonthUsd: "1.84",
      indexingUnpricedCalls: 0,
      retrievalCostMonthUsd: "0.31",
      retrievalUnpricedCalls: 0,
      lastIndexedAt: state.now() - 900,
      modelChange: null,
      configuredModelKey: "openai:text-embedding-3-small:1536",
      configuredModelDiffers: false,
    },
  }),
  AIAuditChainStatus: (state) => ({
    aiAuditChainStatus: {
      signed: true,
      activeKeyId: "k2026-09",
      firstSeq: 1,
      lastSeq: 18422,
      lastHash: "9f2c41ab77e0d1c5",
      sealedThroughSeq: 18400,
      lastVerifiedSeq: 18400,
      lastVerifiedAt: state.now() - 7200,
      lastVerificationStatus: "Verified",
      failedSeq: null,
      detail: "",
      verifying: false,
    },
  }),
  AgentQualityOverview: (state) => ({
    agentQualityOverview: {
      windowDays: 30,
      since: state.now() - 30 * DAY,
      ratingsVisible: true,
      satisfaction: 0.91,
      ratings: 214,
      qualityScore: 0.87,
      agentsScored: 9,
      suiteRuns: 31,
      regressions: 2,
      openRegressions: 1,
      evalSpendMonthUsd: "4.12",
      evalUnpricedCalls: 0,
      monthlyBudgetUsd: "20",
      monthStartedAt: state.now() - 6 * DAY,
      sweepEnabled: true,
      nextSweepHourLocal: 2,
      nextSweepTimezone: "America/Los_Angeles",
      agentsWithCases: 7,
      judgeEnabled: true,
      regressionThreshold: 0.1,
    },
  }),
};

// ---------------------------------------------------------------- agents

const TOOL_BY_NAME = new Map(TOOL_CATALOG.tools.map((tool) => [tool.name, tool]));

function agentOf(state, id) {
  return state.ai.agents.find((agent) => agent.node.id === id) ?? null;
}

function titleOf(name) {
  const words = name.replace(/_/g, " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function egressOf(tool) {
  if (tool.kind === "query") return ["None"];
  const name = tool.name;
  if (/payment|settlement|write_off|refund|payout|charge_card/.test(name)) return ["Money"];
  if (/customer_reply|send_customer|notify_customer|customer_update/.test(name)) return ["CustomerVisible"];
  if (/driver_message|send_driver|notify_driver/.test(name)) return ["DriverVisible"];
  if (/email|tender_to_carrier|send_|edi_/.test(name)) return ["ExternalRecipient"];
  return ["Internal"];
}

function toolRule(tool) {
  const egress = egressOf(tool);
  const destructive = /delete|cancel|void|write_off|transfer|unassign/.test(tool.name);
  const leaves = egress.some((entry) => ["CustomerVisible", "DriverVisible", "ExternalRecipient", "Money"].includes(entry));
  const maxTier = tool.kind === "query" ? "AutoExecute" : destructive ? "Propose" : "AutoExecute";
  return {
    id: tool.name,
    name: tool.name,
    title: titleOf(tool.name),
    kind: tool.kind === "query" ? "Query" : "Action",
    needs: tool.resource ? { resource: tool.resource, operation: tool.operation || "read" } : null,
    scope: "Tenant",
    defaultTier: tool.defaultAutonomyTier || (tool.kind === "query" ? "AutoExecute" : "Propose"),
    maxTier,
    promotableTier: leaves && maxTier === "AutoExecute" ? "ActWithApproval" : maxTier,
    egress,
    leavesOrganization: leaves,
    hasClassify: false,
    hasCondition: false,
    conditionDescription: null,
    personalExemption: false,
    effect: tool.kind === "query" ? "Lookup" : "Change",
    artifact: "",
    reversible: !destructive,
    idempotent: false,
    readsExternal: /inbound|document|email|edi/.test(tool.name) ? "Always" : "Never",
    source: null,
    carriesTaint: false,
    rationale: "",
    explanation: "",
    runsWithoutPerson: null,
  };
}

const TOOL_RULES = TOOL_CATALOG.tools.map(toolRule);

function rosterStat(agent) {
  const runsByDay = agent.runsPerDay.slice(-14);
  const rec = agent.rec ?? { approved: 0, modified: 0, rejected: 0, failed: 0 };
  const decided = rec.approved + rec.modified + rec.rejected;
  return {
    agentId: agent.node.id,
    runsByDay,
    runs: runsByDay.reduce((total, count) => total + count, 0),
    approved: rec.approved,
    modified: rec.modified,
    rejected: rec.rejected,
    failed: rec.failed,
    shadowRecorded: agent.node.shadowMode ? (SHADOW_RECORDED[agent.key] ?? 0) : 0,
    approvalRate: decided ? (rec.approved + rec.modified) / decided : null,
  };
}

function streakTool(agent) {
  return (
    agent.node.toolNames.find((name) => agent.node.toolTiers[name] === "ActWithApproval") ??
    agent.node.toolNames.find((name) => TOOL_BY_NAME.get(name)?.kind === "action") ??
    null
  );
}

function trustOf(state, agent) {
  const tool = agent.rec ? streakTool(agent) : null;
  if (!tool) return [];
  return [
    {
      id: `tt_${agent.key}`,
      organizationId: agent.node.organizationId,
      businessUnitId: agent.node.businessUnitId,
      agentDefinitionId: agent.node.id,
      toolName: tool,
      streak: agent.rec.streak,
      approvals: agent.rec.approved,
      modifications: agent.rec.modified,
      rejections: agent.rec.rejected,
      executionFailures: agent.rec.failed,
      earnedTier: null,
      lastDecisionAt: state.now() - 3600,
      promotedAt: null,
      demotedAt: null,
      createdAt: state.now() - 60 * DAY,
      updatedAt: state.now() - 3600,
    },
  ];
}

function versionsOf(state, agent) {
  const node = agent.node;
  const author = (id, name) => ({ id, name });
  return [
    { id: `agv_${agent.key}_3`, version: node.version, summary: "Raised a tool to Ask first", author: author("usr_sa", "Sarah Alvarez"), createdAt: state.now() - 1 * DAY },
    { id: `agv_${agent.key}_2`, version: node.version - 1, summary: "Added a line to Never", author: author("usr_mr", "Marcus Reed"), createdAt: state.now() - 8 * DAY },
    { id: `agv_${agent.key}_1`, version: node.version - 2, summary: "Created", author: null, createdAt: state.now() - 90 * DAY },
  ].filter((version) => version.version > 0);
}

function accessPreview(state, input) {
  const tools = input.toolNames ?? [];
  const sensitive =
    input.accessMode === "Everyone"
      ? tools.filter((name) => {
          const rule = TOOL_RULES.find((entry) => entry.name === name);
          return rule && rule.kind === "Action" && rule.leavesOrganization;
        })
      : [];
  const granted = new Set(agentOf(state, input.agentId)?.node.accessRoles.map((role) => role.id) ?? []);
  return {
    accessMode: input.accessMode,
    sensitiveTools: sensitive,
    roles: ROLES.map((role) => ({
      role: { ...role, description: "", isSystem: role.name === "Owner" },
      coverage: role.name === "Owner" ? "Full" : "Partial",
      missingResources: [],
      granted: granted.has(role.id),
    })),
  };
}

function scorecardOf(state, agent) {
  const rec = agent.rec ?? { approved: 0, modified: 0, rejected: 0, failed: 0, streak: 0 };
  const runs = agent.runsPerDay.reduce((total, count) => total + count, 0) * 2;
  const decided = rec.approved + rec.modified + rec.rejected;
  return {
    agentDefinitionId: agent.node.id,
    window: "Last30Days",
    since: state.now() - 30 * DAY,
    runs,
    runsFailed: rec.failed,
    exceptions: 0,
    proposals: decided + agent.node.pendingProposals,
    approved: rec.approved,
    modified: rec.modified,
    rejected: rec.rejected,
    pending: agent.node.pendingProposals,
    executed: rec.approved + rec.modified,
    executionFailures: rec.failed,
    autoExecuted: 0,
    approvalRate: decided ? rec.approved / decided : null,
    inputTokens: runs * 2100,
    outputTokens: runs * 380,
    costUsd: (runs * 0.0042).toFixed(2),
    estimatedMinutesSaved: rec.approved * 18,
    byTool: [],
    trend: [],
    toolTrust: trustOf(state, agent).map(({ toolName, streak, approvals, modifications, rejections, executionFailures, earnedTier, lastDecisionAt, promotedAt, demotedAt }) => ({ toolName, streak, approvals, modifications, rejections, executionFailures, earnedTier, lastDecisionAt, promotedAt, demotedAt })),
  };
}

function lintOf(input) {
  const text = input.instructions ?? "";
  const held = new Set(input.toolNames ?? []);
  const findings = [];
  const email = /\bemail\b/i.exec(text);
  if (email && !TOOL_CATALOG.tools.some((tool) => held.has(tool.name) && /email/.test(tool.name))) {
    const tools = TOOL_CATALOG.tools.filter((tool) => /email/.test(tool.name)).map((tool) => tool.name);
    findings.push({ start: email.index, end: email.index + 5, excerpt: email[0], resource: "report_schedule", resourceLabel: "Emails", operation: "send", tools });
  }
  return findings;
}

/** The node an agent's save request makes, merged over what it was. */
function savedNode(state, base, request, id) {
  const roleNames = new Map(ROLES.map((role) => [role.id, role.name]));
  const accessMode = request.accessMode ?? base?.accessMode ?? "Everyone";
  const roleIds = request.accessRoleIds ?? base?.accessRoles.map((role) => role.id) ?? [];
  return {
    ...(base ?? {}),
    ...request,
    id,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    template: request.template ?? base?.template ?? null,
    learningOff: request.learningOff ?? false,
    monthlyBudgetUsd: request.monthlyBudgetUsd == null ? null : String(request.monthlyBudgetUsd),
    preferredProviderId: request.preferredProviderId || null,
    systemKey: base?.systemKey ?? "",
    starters: base?.starters ?? [],
    delegates: (request.delegateIds ?? []).map((delegate) => agentOf(state, delegate)?.node).filter(Boolean).map((node) => ({ id: node.id, name: node.name, icon: node.icon, accent: node.accent, enabled: node.enabled, triggerMode: node.triggerMode })),
    accessMode,
    accessRoles: roleIds.map((roleId) => ({ id: roleId, name: roleNames.get(roleId) ?? roleId })),
    lastRunAt: base?.lastRunAt ?? null,
    nextRunAt: base?.nextRunAt ?? null,
    pendingProposals: base?.pendingProposals ?? 0,
    openRuns: base?.openRuns ?? 0,
    version: (base?.version ?? 0) + 1,
    createdAt: base?.createdAt ?? state.now(),
    updatedAt: state.now(),
  };
}

function dryRunFrames(state, body) {
  const draft = body?.draft ?? {};
  const held = draft.toolNames ?? [];
  const reads = held.filter((name) => TOOL_BY_NAME.get(name)?.kind === "query").slice(0, 2);
  const change = held.find((name) => TOOL_BY_NAME.get(name)?.kind === "action");
  const calls = [...reads, ...(change ? [change] : [])];
  const tier = (name) => (draft.toolTiers ?? {})[name] ?? draft.autonomyCeiling ?? "Propose";
  const outcomeOf = (name) => {
    if (TOOL_BY_NAME.get(name)?.kind !== "action") return "Runs";
    if (draft.simulationMode) return "Simulated";
    if (draft.shadowMode) return "Recorded";
    const effective = tier(name) === "AutoExecute" && draft.autonomyCeiling === "AutoExecute" ? "AutoExecute" : tier(name) === "Propose" || draft.autonomyCeiling === "Propose" ? "Propose" : "ActWithApproval";
    return effective === "AutoExecute" ? "Runs" : effective === "Propose" ? "Propose" : "AskFirst";
  };
  const reply = `Here is what I found for SHP-48302: it picks up tomorrow at 7:00 in Fort Worth and nobody is assigned yet. ${change ? `I would ${titleOf(change).toLowerCase()} next, and a person decides before it happens.` : "I can explain what to do next."}`;
  const frames = [{ event: "accepted", data: { turnId: "turn_mock" }, delay: 200 }];
  calls.forEach((name, index) => {
    frames.push({ event: "tool_started", data: { callId: `c${index}`, name }, delay: 450 });
    frames.push({ event: "tool_finished", data: { callId: `c${index}`, name, failed: false }, delay: 500 });
  });
  for (const word of reply.split(" ")) {
    frames.push({ event: "delta", data: { text: `${word} ` }, delay: 35 });
  }
  frames.push({
    event: "dry_run_steps",
    data: { steps: calls.map((name, index) => ({ callId: `c${index}`, tool: name, outcome: outcomeOf(name) })), reply },
    delay: 120,
  });
  frames.push({ event: "done", data: { reply }, delay: 40 });
  return frames;
}

export const AI_ROUTES = [
  ["GET", /^\/api\/v1\/agent-definitions\/templates\/?$/, () => ({ templates: [] })],
  [
    "POST",
    /^\/api\/v1\/agent-definitions\/dry-run\/?$/,
    (state, _path, body) => ({ __sse: dryRunFrames(state, body) }),
  ],
  [
    "GET",
    /^\/api\/v1\/agent-definitions\/[^/]+\/trust\/?$/,
    (state, path) => {
      const agent = agentOf(state, path.split("/")[4]);
      return { results: agent ? trustOf(state, agent) : [] };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/agent-definitions\/[^/]+\/budget\/?$/,
    (state, path) => {
      const agent = agentOf(state, path.split("/")[4]);
      return {
        monthStart: state.now() - 6 * DAY,
        dayStart: state.now() - (state.now() % DAY),
        spentUsd: agent?.node.monthlyBudgetUsd ? "20.50" : "3.10",
        monthlyBudgetUsd: agent?.node.monthlyBudgetUsd ?? null,
        monthCalls: 412,
        unpricedCalls: 0,
        runsToday: 6,
        dailyRunLimit: agent?.node.dailyRunLimit ?? 0,
        tools: [],
        simulationMode: agent?.node.simulationMode ?? false,
      };
    },
  ],
  [
    "POST",
    /^\/api\/v1\/agent-definitions\/?$/,
    (state, _path, body) => {
      const id = `agd_new${state.ai.agents.length}`;
      const node = savedNode(state, null, body ?? {}, id);
      state.ai.agents.push({ key: id, rec: null, runsPerDay: Array(14).fill(0), node });
      return node;
    },
  ],
  [
    "PUT",
    /^\/api\/v1\/agent-definitions\/[^/]+\/?$/,
    (state, path, body) => {
      const agent = agentOf(state, path.split("/")[4]);
      if (!agent) return { __status: 404, body: { message: "Agent not found" } };
      if (body?.version !== undefined && body.version !== agent.node.version) {
        return { __status: 409, body: { type: "version_mismatch", message: "Someone else saved this agent" } };
      }
      agent.node = savedNode(state, agent.node, body ?? {}, agent.node.id);
      return agent.node;
    },
  ],
  [
    "DELETE",
    /^\/api\/v1\/agent-definitions\/[^/]+\/?$/,
    (state, path) => {
      const id = path.split("/")[4];
      state.ai.agents = state.ai.agents.filter((agent) => agent.node.id !== id);
      return {};
    },
  ],
  ["GET", /^\/api\/v1\/ai-providers\/catalog\/?$/, () => PROVIDER_CATALOG],
  ["GET", /^\/api\/v1\/agent-extensions\/catalog\/?$/, () => EXTENSION_CATALOG],
  ["GET", /^\/api\/v1\/agent-definitions\/tools\/?$/, () => TOOL_CATALOG],
  ["GET", /^\/api\/v1\/agent-definitions\/event-kinds\/?$/, () => EVENT_KINDS],
];
