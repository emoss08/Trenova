// The agent a question is best put to, as `agentruntime.RecommendAgent` works
// it out: the question ranks the tools the agents hold, only matches count (a
// tool scoring under half the best match is cut), an agent scores 1/rank for
// each matched tool it holds, and ties go to the agent the question would go to
// now, then to the agent holding fewer tools, then by name. The server ranks
// with keyword and embedding fusion; here the keywords alone stand in, read
// from each tool's description and the words people use for it.
//
// Scenario `agentMatch`: match (the default) | none (no agent fits) | hold
// (the request is left open, as a slow match is, for the "finding" state).

const HOLD = { __hold: true };

const TOOLS = {
  create_shipment: "Create a shipment load order from a tender or email booking",
  tender_shipment: "Tender a shipment load to a carrier for coverage",
  search_carriers: "Search carriers by lane equipment authority to cover a load",
  file_document: "File a document such as a POD bill of lading or lumper receipt on a shipment",
  waive_detention: "Waive or approve detention charges billed on an invoice for a stop",
  queue_carrier_pay: "Queue carrier pay settlement for a delivered load invoice",
  shipment_eta: "Shipment ETA arrival time status for a customer",
  draft_reply: "Draft an email reply to a customer about status",
  get_tractor: "Retrieve one tractor truck power unit",
  get_trailer: "Retrieve one trailer",
  get_worker: "Retrieve one worker driver",
  get_location: "Retrieve one location facility",
  get_carrier: "Retrieve one carrier",
  get_order: "Retrieve one order",
  list_tractors: "List tractors trucks by status",
  get_shipment: "Retrieve one shipment load",
};

const STOP_WORDS = new Set([
  "a",
  "an",
  "and",
  "are",
  "can",
  "do",
  "does",
  "for",
  "from",
  "has",
  "have",
  "how",
  "i",
  "if",
  "in",
  "is",
  "it",
  "me",
  "my",
  "of",
  "on",
  "or",
  "our",
  "please",
  "should",
  "the",
  "this",
  "to",
  "was",
  "we",
  "what",
  "when",
  "which",
  "who",
  "why",
  "will",
  "with",
  "you",
]);

function terms(text) {
  return new Set(
    text
      .toLowerCase()
      .split(/[^a-z0-9]+/u)
      .filter((word) => word.length > 1 && !STOP_WORDS.has(word))
      .map((word) => (word.length > 3 && word.endsWith("s") ? word.slice(0, -1) : word)),
  );
}

function rankTools(text, names) {
  const wanted = terms(text);
  const scored = names
    .map((name) => {
      const nameWords = terms(name.replaceAll("_", " "));
      const body = terms(TOOLS[name] ?? "");
      let score = 0;
      for (const word of wanted) {
        if (nameWords.has(word)) score += 4;
        else if (body.has(word)) score += 2;
      }
      return { name, score };
    })
    .filter((tool) => tool.score > 0)
    .sort((a, b) => b.score - a.score || a.name.localeCompare(b.name));
  if (scored.length === 0) return [];
  const floor = scored[0].score / 2;

  return scored
    .filter((tool) => tool.score >= floor)
    .slice(0, 8)
    .map((tool) => tool.name);
}

export function recommendAgent(state, variables, agents) {
  if (state.scenario.agentMatch === "hold") return HOLD;
  if (state.scenario.agentMatch === "none") return { myAgentRecommendation: null };

  const text = String(variables.input?.text ?? "").trim();
  const currentId = variables.input?.currentAgentId ?? "";
  if (text === "") return { myAgentRecommendation: null };

  const held = new Set(agents.flatMap((agent) => agent.toolNames ?? []));
  const ranked = rankTools(text, [...held]);
  if (ranked.length === 0) return { myAgentRecommendation: null };

  const candidates = agents
    .map((agent) => {
      const tools = ranked.filter((name) => (agent.toolNames ?? []).includes(name));
      const score = tools.reduce((sum, name) => sum + 1 / (ranked.indexOf(name) + 1), 0);
      return { agent, tools, score, current: agent.id === currentId };
    })
    .filter((candidate) => candidate.score > 0)
    .sort(
      (a, b) =>
        b.score - a.score ||
        Number(b.current) - Number(a.current) ||
        (a.agent.toolNames?.length ?? 0) - (b.agent.toolNames?.length ?? 0) ||
        a.agent.name.localeCompare(b.agent.name),
    );
  const best = candidates[0];
  if (!best) return { myAgentRecommendation: null };

  return {
    myAgentRecommendation: {
      agent: best.agent,
      current: best.current,
      tools: best.tools.slice(0, 3),
    },
  };
}
