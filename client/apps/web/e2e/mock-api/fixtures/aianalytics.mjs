// AI analytics: a deterministic month of finished AI work, one row per run, turn or call, in the
// shape the server's analytics refresh writes (ai_work_units and the tables beside it). The
// handler aggregates these the way aianalyticsservice does, so filters, sheets and routing move
// together. The figures are the design handoff's: six models on seven providers, Llama 3.3 70B
// split between Groq and a workstation vLLM, and four provider events on the trend.

const DAY = 86400;
const HOUR = 3600;

export const ANALYTICS_PROVIDERS = {
  aip_groq: {
    id: "aip_groq",
    name: "Groq",
    kind: "OpenAIChat",
    baseUrl: "https://api.groq.com/openai/v1",
    enabled: true,
    model: "llama-3.3-70b-versatile",
    firstToken: 180,
    timeout: 60,
  },
  aip_vllm: {
    id: "aip_vllm",
    name: "Workstation vLLM",
    kind: "OpenAIChat",
    baseUrl: "http://localhost:8000/v1",
    enabled: true,
    model: "meta-llama/Llama-3.3-70B-Instruct",
    firstToken: 910,
    timeout: 60,
  },
  aip_openai: {
    id: "aip_openai",
    name: "OpenAI",
    kind: "OpenAIResponses",
    baseUrl: "",
    enabled: true,
    model: "gpt-5.1-mini",
    firstToken: 420,
    timeout: 60,
  },
  aip_ollama: {
    id: "aip_ollama",
    name: "Local Ollama",
    kind: "Ollama",
    baseUrl: "http://localhost:11434",
    enabled: true,
    model: "qwen2.5:14b",
    firstToken: 380,
    timeout: 60,
  },
  aip_azure: {
    id: "aip_azure",
    name: "Azure OpenAI",
    kind: "OpenAIChat",
    baseUrl: "https://trenova.openai.azure.com/openai/v1",
    enabled: true,
    model: "gpt-5.1",
    firstToken: 650,
    timeout: 60,
  },
  aip_bedrock: {
    id: "aip_bedrock",
    name: "AWS Bedrock",
    kind: "AnthropicMessages",
    baseUrl: "",
    enabled: true,
    model: "anthropic.claude-sonnet-5",
    firstToken: 687,
    timeout: 60,
  },
  aip_gemini: {
    id: "aip_gemini",
    name: "Google Gemini",
    kind: "OpenAIChat",
    baseUrl: "https://generativelanguage.googleapis.com/v1beta/openai",
    enabled: false,
    model: "gemini-3-flash",
    firstToken: 300,
    timeout: 60,
  },
};

/** The order the router tries providers in; a task goes to the first that serves it. */
export const ROUTING_ORDER = [
  "aip_ollama",
  "aip_openai",
  "aip_vllm",
  "aip_groq",
  "aip_azure",
  "aip_bedrock",
  "aip_gemini",
];

export const INITIAL_ROUTING = {
  aip_openai: ["DocumentClassification", "ScopeClassification", "QueryCompose"],
  aip_ollama: [
    "DocumentClassification",
    "ScopeClassification",
    "OperationalInsights",
    "DailyBriefing",
  ],
  aip_vllm: ["AssistantChat", "DocumentExtraction", "FormulaAssistant"],
  aip_groq: ["OperationalInsights", "FormulaAssistant"],
  aip_azure: ["DocumentExtraction", "BillingDiagnosis"],
  aip_bedrock: ["AssistantChat", "BillingDiagnosis"],
  aip_gemini: ["DailyBriefing"],
};

export const MODELS = {
  "llama-3.3-70b": {
    median: 6200,
    sigma: 0.55,
    toolFail: 0.045,
    modelShare: 0.35,
    cost: null,
    accept: 0.61,
    decided: 1215,
    helpful: 0.71,
  },
  "gpt-5.1-mini": {
    median: 4100,
    sigma: 0.45,
    toolFail: 0.026,
    modelShare: 0.2,
    cost: 0.004,
    accept: 0.7,
    decided: 1085,
    helpful: 0.84,
  },
  "qwen-2.5-14b": {
    median: 5700,
    sigma: 0.5,
    toolFail: 0.09,
    modelShare: 0.78,
    cost: null,
    accept: 0.52,
    decided: 553,
    helpful: 0.63,
  },
  "gpt-5.1": {
    median: 10200,
    sigma: 0.42,
    toolFail: 0.021,
    modelShare: 0.16,
    cost: 0.031,
    accept: 0.78,
    decided: 434,
    helpful: 0.9,
  },
  "claude-sonnet-5": {
    median: 13000,
    sigma: 0.4,
    toolFail: 0.018,
    modelShare: 0.14,
    cost: 0.052,
    accept: 0.84,
    decided: 164,
    helpful: 0.93,
  },
  "gemini-3-flash": {
    median: 4200,
    sigma: 0.45,
    toolFail: 0.035,
    modelShare: 0.3,
    cost: 0.006,
    accept: 0.69,
    decided: 106,
    helpful: 0.8,
  },
};

/** How many runs of each task went where, and how many of them completed. */
const ROUTES = [
  {
    task: "AssistantChat",
    model: "llama-3.3-70b",
    provider: "aip_vllm",
    runs: 900,
    rate: 0.72,
    agents: ["dispatch", "billing", "compliance", "recv"],
  },
  {
    task: "AssistantChat",
    model: "claude-sonnet-5",
    provider: "aip_bedrock",
    runs: 300,
    rate: 0.96,
    agents: ["dispatch", "billing", "recv"],
    until: 7,
  },
  {
    task: "BillingDiagnosis",
    model: "gpt-5.1",
    provider: "aip_azure",
    runs: 400,
    rate: 0.93,
    agents: ["billev"],
  },
  {
    task: "BillingDiagnosis",
    model: "claude-sonnet-5",
    provider: "aip_bedrock",
    runs: 168,
    rate: 0.95,
    agents: ["billev"],
    until: 7,
  },
  {
    task: "DocumentExtraction",
    model: "llama-3.3-70b",
    provider: "aip_vllm",
    runs: 700,
    rate: 0.7,
    agents: ["inbox"],
  },
  {
    task: "DocumentExtraction",
    model: "gpt-5.1",
    provider: "aip_azure",
    runs: 840,
    rate: 0.95,
    agents: ["inbox"],
  },
  {
    task: "DocumentClassification",
    model: "gpt-5.1-mini",
    provider: "aip_openai",
    runs: 900,
    rate: 0.97,
    agents: ["inbox"],
  },
  {
    task: "DocumentClassification",
    model: "qwen-2.5-14b",
    provider: "aip_ollama",
    runs: 600,
    rate: 0.89,
    agents: ["inbox"],
  },
  {
    task: "ScopeClassification",
    model: "gpt-5.1-mini",
    provider: "aip_openai",
    runs: 1000,
    rate: 0.97,
    agents: ["dispatch", "help"],
  },
  {
    task: "QueryCompose",
    model: "gpt-5.1-mini",
    provider: "aip_openai",
    runs: 1200,
    rate: 0.82,
    agents: ["report", "dispatch"],
  },
  {
    task: "ScopeClassification",
    model: "qwen-2.5-14b",
    provider: "aip_ollama",
    runs: 500,
    rate: 0.93,
    agents: ["dispatch", "help"],
  },
  {
    task: "OperationalInsights",
    model: "llama-3.3-70b",
    provider: "aip_groq",
    runs: 1200,
    rate: 0.86,
    agents: ["coverage", "digest"],
  },
  {
    task: "OperationalInsights",
    model: "qwen-2.5-14b",
    provider: "aip_ollama",
    runs: 300,
    rate: 0.74,
    agents: ["coverage"],
  },
  {
    task: "DailyBriefing",
    model: "qwen-2.5-14b",
    provider: "aip_ollama",
    runs: 180,
    rate: 0.81,
    agents: ["digest"],
  },
  {
    task: "DailyBriefing",
    model: "gemini-3-flash",
    provider: "aip_gemini",
    runs: 304,
    rate: 0.92,
    agents: ["digest"],
    until: 6,
  },
  {
    task: "FormulaAssistant",
    model: "llama-3.3-70b",
    provider: "aip_vllm",
    runs: 670,
    rate: 0.81,
    agents: ["billing"],
  },
];

const TOOLS_BY_TASK = {
  AssistantChat: [
    "assign_driver",
    "search_drivers",
    "lookup_rate",
    "get_hos_status",
    "update_load_status",
    "run_report",
  ],
  BillingDiagnosis: ["release_billing_hold", "lookup_rate", "run_report"],
  DocumentExtraction: ["extract_document_fields"],
  DocumentClassification: ["extract_document_fields"],
  QueryCompose: ["run_report"],
  ScopeClassification: ["search_drivers"],
  OperationalInsights: ["run_report"],
  DailyBriefing: ["run_report"],
  FormulaAssistant: ["lookup_rate"],
};

const CALLS_BY_TASK = {
  AssistantChat: 6,
  BillingDiagnosis: 5,
  DocumentExtraction: 2,
  DocumentClassification: 0.3,
  QueryCompose: 0.4,
  ScopeClassification: 0.2,
  OperationalInsights: 1,
  DailyBriefing: 1,
  FormulaAssistant: 2,
};

const MODEL_CAUSES = [
  ["WrongArguments", 0.72],
  ["RepeatedCall", 0.17],
  ["ToolNotHeld", 0.11],
];
const TOOL_CAUSES = [
  ["ToolTimedOut", 0.42],
  ["BlockedByRule", 0.33],
  ["ToolError", 0.25],
];

export const FAILURE_MESSAGES = {
  WrongArguments: {
    assign_driver: '"driver_id" must be a string — got 1042',
    run_report: '"date_range.start" is not a date: "last tuesday"',
    release_billing_hold: '"hold_id" is required',
    lookup_rate: '"lane.origin" must be a location id, got "Dallas"',
    extract_document_fields: '"fields" must list at least one field',
    update_load_status: '"status" must be one of InTransit, Delivered, got "delivered "',
    get_hos_status: '"worker_id" is required',
    search_drivers: '"filters.available_on" is not a date',
  },
  ToolTimedOut: "The tool did not answer within 30 seconds",
  RepeatedCall: "The same call was already made this turn",
  BlockedByRule: "The tool rule allows this only with a person's approval",
  ToolError: "record not found",
  ToolNotHeld: "Tool is not available to this agent",
};

function mulberry32(seed) {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function normal(random) {
  const u = Math.max(random(), 1e-9);
  const v = random();
  return Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v);
}

function pick(random, weighted) {
  let roll = random();
  for (const [value, weight] of weighted) {
    roll -= weight;
    if (roll <= 0) return value;
  }
  return weighted[weighted.length - 1][0];
}

/** The workstation stopped answering this morning, and limped for a day before it restarted 12 days ago. */
function rateOn(route, daysAgo) {
  if (route.provider === "aip_vllm" && daysAgo === 0) return route.rate - 0.2;
  if (route.provider === "aip_vllm" && daysAgo === 13) return route.rate - 0.12;
  return route.rate;
}

/** How much faster or slower a provider serves the same model: the workstation is slow, Groq fast. */
const SPEED = { aip_vllm: 1.9, aip_groq: 0.55 };

/** Each provider's share of model calls that fail before an answer, and what kind they are. */
const PROVIDER_ERRORS = {
  aip_vllm: {
    rate: 0.06,
    kinds: [
      ["TimedOut", 0.62],
      ["CouldNotConnect", 0.32],
      ["ServerError", 0.06],
    ],
  },
  aip_ollama: {
    rate: 0.012,
    kinds: [
      ["CouldNotConnect", 0.4],
      ["TimedOut", 0.6],
    ],
  },
  aip_groq: {
    rate: 0.024,
    kinds: [
      ["RateLimited", 0.9],
      ["ServerError", 0.07],
      ["TimedOut", 0.03],
    ],
  },
  aip_openai: {
    rate: 0.008,
    kinds: [
      ["RateLimited", 0.55],
      ["ServerError", 0.27],
      ["TimedOut", 0.18],
    ],
  },
  aip_azure: {
    rate: 0.006,
    kinds: [
      ["ServerError", 0.45],
      ["RateLimited", 0.33],
      ["TimedOut", 0.22],
    ],
  },
  aip_bedrock: { rate: 0.004, kinds: [["RateLimited", 1]] },
  aip_gemini: { rate: 0.006, kinds: [["RateLimited", 1]] },
};

export function buildAnalyticsUnits(anchor) {
  const random = mulberry32(20261013);
  const units = [];
  const errors = [];
  let serial = 0;
  for (const route of ROUTES) {
    const model = MODELS[route.model];
    const lastDay = route.until ?? 0;
    const days = 30 - lastDay;
    for (let index = 0; index < route.runs; index += 1) {
      const daysAgo = lastDay + Math.floor((index / route.runs) * days);
      const finishedAt = anchor - daysAgo * DAY - Math.floor(random() * DAY * 0.9);
      const completed = (index * 0.6180339887 + route.runs * 0.137) % 1 < rateOn(route, daysAgo);
      let outcome = "Completed";
      if (!completed) {
        const roll = random();
        outcome =
          route.provider === "aip_vllm"
            ? roll < 0.15
              ? "HandedOff"
              : roll < 0.22
                ? "Stopped"
                : "Failed"
            : roll < 0.45
              ? "HandedOff"
              : roll < 0.6
                ? "Stopped"
                : "Failed";
      }
      let duration =
        model.median * (SPEED[route.provider] ?? 1) * Math.exp(model.sigma * normal(random));
      if (outcome === "Failed" && route.provider === "aip_vllm") duration = 60000 + random() * 9000;
      else if (outcome === "Failed") duration *= 1.8;
      else if (outcome === "Stopped") duration *= 4.5;
      else if (outcome === "HandedOff") duration *= 0.8;

      const calls = CALLS_BY_TASK[route.task];
      const toolNames = TOOLS_BY_TASK[route.task];
      const toolCount =
        calls >= 1
          ? Math.max(0, Math.round(calls + normal(random) * calls * 0.4))
          : random() < calls
            ? 1
            : 0;
      const tools = new Map();
      for (let call = 0; call < toolCount; call += 1) {
        const name = toolNames[Math.floor(random() * toolNames.length)];
        const entry = tools.get(name) ?? { calls: 0, failures: [] };
        entry.calls += 1;
        const failRate =
          model.toolFail *
          (name === "assign_driver" || name === "extract_document_fields"
            ? 1.35
            : name === "search_drivers"
              ? 0.7
              : 1);
        if (random() < failRate) {
          const cause =
            random() < model.modelShare ? pick(random, MODEL_CAUSES) : pick(random, TOOL_CAUSES);
          entry.failures.push(cause);
        }
        tools.set(name, entry);
      }

      const provider = ANALYTICS_PROVIDERS[route.provider];
      const firstError = errors.length;
      const modelCalls = 1 + Math.ceil(toolCount / 2);
      let failedCalls = 0;
      const profile = PROVIDER_ERRORS[route.provider];
      const providerErrorRate =
        route.provider === "aip_vllm" && daysAgo === 0 ? 0.45 : profile.rate;
      for (let call = 0; call < modelCalls; call += 1) {
        if (random() < providerErrorRate) {
          failedCalls += 1;
          errors.push({
            providerId: route.provider,
            model: route.model,
            task: route.task,
            at: finishedAt,
            kind: pick(random, profile.kinds),
            fell: outcome !== "Failed" && random() < 0.85,
          });
        }
      }
      if (route.provider === "aip_bedrock" && daysAgo === lastDay) {
        errors.push({
          providerId: "aip_bedrock",
          model: route.model,
          task: route.task,
          at: anchor - lastDay * DAY + 2 * HOUR,
          kind: "AccessDenied",
          fell: true,
        });
      }

      serial += 1;
      const decisionRoll = random();
      const decided = decisionRoll < model.decided / modelTotal(route.model);
      let decision = null;
      if (decided) {
        const roll = random();
        decision =
          roll < model.accept
            ? "Accepted"
            : roll < model.accept + (1 - model.accept) * 0.55
              ? "Modified"
              : "Rejected";
      }
      const rated = route.task === "AssistantChat" && random() < 0.35;
      units.push({
        id: `unit_${serial}`,
        source:
          route.task === "AssistantChat"
            ? "AssistantTurn"
            : route.task === "BillingDiagnosis"
              ? "AgentRun"
              : "Call",
        modelKey: route.model,
        providerId: route.provider,
        task: route.task,
        agentId: `agd_${route.agents[Math.floor(random() * route.agents.length)]}`,
        outcome,
        finishedAt,
        durationMs: Math.round(duration),
        firstTokenMs: Math.round(provider.firstToken * Math.exp(0.35 * normal(random))),
        modelCalls,
        failedCalls,
        cost:
          model.cost === null ? null : model.cost * (0.6 + random() * 0.8) * (completed ? 1 : 0.7),
        tools: [...tools.entries()].map(([name, entry]) => ({
          name,
          calls: entry.calls,
          failures: entry.failures,
        })),
        decision,
        rating: rated ? (random() < model.helpful ? 1 : -1) : null,
      });
      const agentId = units[units.length - 1].agentId;
      for (const error of errors.slice(firstError)) error.agentId = agentId;
    }
  }
  units.sort((left, right) => left.finishedAt - right.finishedAt);
  return { units, errors };
}

function modelTotal(model) {
  return ROUTES.filter((route) => route.model === model).reduce(
    (sum, route) => sum + route.runs,
    0,
  );
}

/** What the provider history and the hourly health say happened, as the server marks them. */
export function analyticsEvents(anchor) {
  const at = (days, hours) => anchor - days * DAY + hours * HOUR - (anchor % HOUR);
  return [
    {
      at: at(12, -10),
      kind: "Recovered",
      providerId: "aip_vllm",
    },
    {
      at: at(7, 2),
      kind: "LostAccess",
      providerId: "aip_bedrock",
    },
    {
      at: at(6, 1),
      kind: "TurnedOff",
      providerId: "aip_gemini",
    },
    {
      at: at(0, -11),
      kind: "StoppedAnswering",
      providerId: "aip_vllm",
    },
  ];
}

export function createAnalyticsState(anchor) {
  const { units, errors } = buildAnalyticsUnits(anchor);
  const routing = Object.fromEntries(
    Object.entries(INITIAL_ROUTING).map(([id, tasks]) => [id, [...tasks]]),
  );
  return { units, errors, routing, refreshedAt: anchor - 120 };
}
