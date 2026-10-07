// AI control: the GraphQL operations and REST routes /admin/agent-control sends.
import {
  EVENT_KINDS,
  EXTENSION_CATALOG,
  PROVIDER_CATALOG,
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

export const AI_ROUTES = [
  ["GET", /^\/api\/v1\/ai-providers\/catalog\/?$/, () => PROVIDER_CATALOG],
  ["GET", /^\/api\/v1\/agent-extensions\/catalog\/?$/, () => EXTENSION_CATALOG],
  ["GET", /^\/api\/v1\/agent-definitions\/tools\/?$/, () => TOOL_CATALOG],
  ["GET", /^\/api\/v1\/agent-definitions\/event-kinds\/?$/, () => EVENT_KINDS],
];
