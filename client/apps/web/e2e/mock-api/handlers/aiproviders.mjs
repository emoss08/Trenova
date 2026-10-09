// AI providers: the chain, its editor and the routing grid on /admin/agent-control.
import { PROVIDER_CATALOG } from "../fixtures/aicontrol.mjs";

const DAY = 86400;
const TRUSTED_TASKS = new Set(
  PROVIDER_CATALOG.tasks.filter((task) => task.requiresTrust).map((task) => task.task),
);
const ALL_TASKS = PROVIDER_CATALOG.tasks.map((task) => task.task);
const KEYED_KINDS = new Set(
  PROVIDER_CATALOG.kinds.filter((kind) => kind.requiresApiKey).map((kind) => kind.kind),
);

const MODELS = {
  AnthropicMessages: [
    ["claude-sonnet-5", 200000, null, false, ["3", "15"]],
    ["claude-haiku-5", 200000, null, false, ["0.8", "4"]],
    ["claude-opus-5", 200000, null, false, ["15", "75"]],
  ],
  OpenAIResponses: [
    ["gpt-5.1", 400000, null, false, ["1.25", "10"]],
    ["gpt-5.1-mini", 400000, null, false, ["0.25", "2"]],
    ["text-embedding-3-large", null, null, true, ["0.13", null]],
  ],
  Ollama: [
    ["qwen2.5:14b", 32768, 9.0e9, false, null, true],
    ["llama3.3:70b", 131072, 43e9, false, null],
    ["nomic-embed-text", null, 274e6, true, null],
  ],
  OpenAIChat: [
    ["meta-llama/Llama-3.3-70B-Instruct", 131072, null, false, null],
    ["Qwen/Qwen2.5-32B-Instruct", 32768, null, false, null],
  ],
};

function nodeOf(state, id) {
  const provider = state.ai.providers.find((entry) => entry.node.id === id);
  if (!provider) throw new Error(`No provider ${id}`);
  return provider;
}

function bump(state, provider) {
  provider.node.version += 1;
  provider.node.updatedAt = state.now();
}

function keyInfo(state, key) {
  const prefix = /^(sk-ant-|sk-proj-|sk-or-|sk-|gsk_|AIza|pa-)/.exec(key)?.[1] ?? "";
  return {
    prefix,
    lastFour: key.slice(-4),
    addedAt: state.now(),
    addedBy: { id: "usr_mock", name: "Morgan Lee" },
    lastUsedAt: null,
    previousKeyExpiresAt: null,
  };
}

function servable(node, task) {
  return (
    node.enabled &&
    node.tasks.includes(task) &&
    (!TRUSTED_TASKS.has(task) || node.trusted) &&
    (task !== "Embedding" || (node.kind !== "AnthropicMessages" && [768, 1024, 1536].includes(node.embeddingDimensions)))
  );
}

function routeFor(nodes, task) {
  const ordered = [...nodes].sort((a, b) => a.priority - b.priority);
  return ordered.find((node) => servable(node, task)) ?? null;
}

function routePreview(state, draft) {
  const nodes = state.ai.providers.map((provider) => provider.node);
  const draftNode = {
    id: draft.id ?? "__draft",
    name: draft.name,
    kind: draft.kind,
    tasks: draft.tasks,
    priority: draft.priority,
    embeddingDimensions: draft.embeddingDimensions ?? null,
    trusted: draft.trusted,
    enabled: draft.enabled,
  };
  const after = [...nodes.filter((node) => node.id !== draftNode.id), draftNode];
  const choice = (node, isDraft) =>
    node ? { providerId: isDraft && !draft.id ? null : node.id, name: node.name, draft: isDraft } : null;
  return ALL_TASKS.map((task) => {
    const before = routeFor(nodes, task);
    const next = routeFor(after, task);
    const beforeId = before?.id ?? null;
    const afterId = next?.id ?? null;
    return {
      task,
      before: choice(before, false),
      after: choice(next, next === draftNode),
      changed: beforeId !== afterId,
    };
  });
}

function providerDays(state, provider) {
  if (!provider?.week) return null;
  const today = state.now() - (state.now() % DAY);
  let seed = provider.node.id.length * 7;
  const random = () => (seed = (seed * 9301 + 49297) % 233280) / 233280;
  const base = provider.week.calls / 7;
  const failures = provider.failing
    ? [0, 0, 0, 0, 0, Math.round(provider.week.failed * 0.66), provider.week.failed - Math.round(provider.week.failed * 0.66)]
    : [0, 0, 0, 0, Math.ceil(provider.week.failed * 0.6), Math.floor(provider.week.failed * 0.4), 0];
  return Array.from({ length: 7 }, (_, index) => ({
    day: new Date((today - (6 - index) * DAY) * 1000).toISOString().slice(0, 10),
    calls: Math.round(base * (0.6 + random() * 0.8)),
    failed: failures[index],
  }));
}

function draftTest(state, input) {
  const { endpoint, model } = input;
  const stored = endpoint.providerId ? nodeOf(state, endpoint.providerId).node : null;
  const local = !KEYED_KINDS.has(endpoint.kind);
  const base = endpoint.baseUrl || (endpoint.kind === "Ollama" ? "http://localhost:11434" : "");
  const fail = (message, detail, hint) => ({
    success: false,
    message,
    detail,
    hint,
    modelIdentifier: "",
    schemaHonoured: false,
    latencyMs: 0,
  });
  if (/:8000/.test(base)) {
    return fail(
      "Connection refused",
      "dial tcp 127.0.0.1:8000: connect: connection refused",
      "Is the server running? vLLM listens on :8000 by default.",
    );
  }
  if (!local && !endpoint.apiKey && !stored?.hasApiKey) {
    return fail("No API key", "401 · missing credentials", "Add a key below, then test again.");
  }
  return {
    success: true,
    message: "Connected",
    detail: `${model} answered a 12-token probe`,
    hint: "",
    modelIdentifier: model,
    schemaHonoured: true,
    latencyMs: local ? 386 : 241,
  };
}

function saveRequestToNode(state, body, existing) {
  const node = existing ?? {
    id: `aip_${Math.random().toString(36).slice(2, 10)}`,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    hasApiKey: false,
    lastTest: null,
    monthSpendUsd: "0",
    apiKey: null,
    version: 0,
    createdAt: state.now(),
  };
  const key = typeof body.apiKey === "string" ? body.apiKey.trim() : undefined;
  if (key) {
    const previous = node.apiKey;
    node.apiKey = keyInfo(state, key);
    if (previous && body.keepPreviousKey) node.apiKey.previousKeyExpiresAt = state.now() + DAY;
    node.hasApiKey = true;
  } else if (key === "" && !existing) {
    node.hasApiKey = false;
    node.apiKey = null;
  }
  Object.assign(node, {
    name: body.name,
    description: body.description ?? "",
    kind: body.kind,
    baseUrl: body.baseUrl ?? "",
    model: body.model,
    allowPrivateNetwork: !!body.allowPrivateNetwork,
    structuredOutputMode: body.structuredOutputMode,
    reasoningEffort: body.reasoningEffort ?? "Off",
    thinkingStyle: body.thinkingStyle ?? "Auto",
    extraBody: body.extraBody ?? null,
    inputCostPerMillion: body.inputCostPerMillion == null ? null : String(body.inputCostPerMillion),
    outputCostPerMillion: body.outputCostPerMillion == null ? null : String(body.outputCostPerMillion),
    maxTokens: body.maxTokens ?? 8192,
    tasks: body.tasks ?? [],
    priority: body.priority ?? 100,
    embeddingDimensions: body.embeddingDimensions ?? null,
    embeddingInputStyle: body.embeddingInputStyle ?? "None",
    trusted: !!body.trusted,
    enabled: !!body.enabled,
    timeoutSeconds: body.timeoutSeconds ?? 60,
    maxConcurrent: body.maxConcurrent ?? 8,
    monthlyCapUsd: body.monthlyCapUsd == null ? null : String(body.monthlyCapUsd),
    onCap: body.onCap ?? "Next",
    version: node.version + 1,
    updatedAt: state.now(),
  });
  return node;
}

export const PROVIDER_HANDLERS = {
  AIProviderDetail: (state, v) => ({
    aiProvider: state.ai.providers.find((provider) => provider.node.id === v.id)?.node ?? null,
  }),
  AIProviderLimits: (state) => ({
    aiProviders: { edges: state.ai.providers.map((provider) => ({ node: provider.node })) },
  }),
  AIRoutePreview: (state, v) => ({ aiRoutePreview: routePreview(state, v.draft) }),
  AIProviderModels: (_state, v) => ({
    aiProviderModels: (MODELS[v.input.kind] ?? []).map(([id, contextWindow, sizeBytes, embedding, price, loaded]) => ({
      id,
      displayName: id,
      contextWindow,
      sizeBytes,
      loaded: !!loaded,
      embedding,
      inputCostPerMillion: price?.[0] ?? null,
      outputCostPerMillion: price?.[1] ?? null,
    })),
  }),
  AIProviderUsageDaily: (state, v) => ({
    aiUsageDaily: providerDays(state, state.ai.providers.find((provider) => provider.node.id === v.providerId)) ?? [],
  }),
  TestAIProviderDraft: (state, v) => ({ testAIProviderDraft: draftTest(state, v.input) }),
  ReorderAIProviders: (state, v) => {
    v.ids.forEach((id, index) => {
      const provider = nodeOf(state, id);
      if (provider.node.priority !== (index + 1) * 10) {
        provider.node.priority = (index + 1) * 10;
        bump(state, provider);
      }
    });
    state.ai.providers.sort((a, b) => a.node.priority - b.node.priority);
    return { reorderAIProviders: state.ai.providers.map((provider) => provider.node) };
  },
  PatchAIProvider: (state, v) => {
    const provider = nodeOf(state, v.id);
    const { apiKey, ...rest } = v.input;
    for (const [field, value] of Object.entries(rest)) {
      if (value !== undefined) provider.node[field] = value;
    }
    if (typeof apiKey === "string" && apiKey.trim()) {
      provider.node.apiKey = keyInfo(state, apiKey.trim());
      provider.node.hasApiKey = true;
    }
    bump(state, provider);
    return { patchAIProvider: provider.node };
  },
};

export const PROVIDER_ROUTES = [
  [
    "POST",
    /^\/api\/v1\/ai-providers\/?$/,
    (state, _path, body) => {
      const node = saveRequestToNode(state, body, null);
      state.ai.providers.push({ key: node.id, week: null, failing: 0, node });
      return node;
    },
  ],
  [
    "POST",
    /^\/api\/v1\/ai-providers\/[^/]+\/test\/?$/,
    (state, path) => {
      const provider = nodeOf(state, path.split("/")[4]);
      const result = draftTest(state, {
        endpoint: {
          providerId: provider.node.id,
          kind: provider.node.kind,
          baseUrl: provider.node.baseUrl,
          apiKey: null,
          allowPrivateNetwork: provider.node.allowPrivateNetwork,
        },
        model: provider.node.model,
      });
      provider.node.lastTest = {
        success: result.success,
        message: result.message,
        modelIdentifier: result.modelIdentifier,
        schemaHonoured: result.schemaHonoured,
        latencyMs: result.latencyMs,
        detail: result.detail,
        testedAt: state.now(),
      };
      if (result.success) provider.failing = 0;
      return result;
    },
  ],
  [
    "PUT",
    /^\/api\/v1\/ai-providers\/[^/]+\/?$/,
    (state, path, body) => {
      const provider = nodeOf(state, path.split("/")[4]);
      if (body.version !== provider.node.version) {
        return { __status: 409, body: { type: "version_mismatch", message: "Someone else saved this provider" } };
      }
      saveRequestToNode(state, body, provider.node);
      return provider.node;
    },
  ],
  [
    "DELETE",
    /^\/api\/v1\/ai-providers\/[^/]+\/?$/,
    (state, path) => {
      const id = path.split("/")[4];
      state.ai.providers = state.ai.providers.filter((provider) => provider.node.id !== id);
      return {};
    },
  ],
];
