// How well the person's agents fit a question, as `agentruntime.MatchAgent`
// works it out: the question ranks the tools, only matches count (a tool
// scoring under half the best match is cut), and only an agent holding the
// best match is suggested. When no agent holds any tool that matched, none of
// them can help (NoAgent). The agent the question would go to now is kept when
// it holds the best match; otherwise the holder scoring 1/rank for each matched
// tool it holds wins, then the one holding fewer tools, then by name. The
// server ranks with keyword and embedding fusion; here the keywords alone
// stand in, read from each tool's description and the words people use for it.
//
// Scenario `agentMatch`: match (the default) | none (no tool matches) |
// nobody (tools match but no agent holds them) | hold (the request is left
// open, as a slow match is, for the "finding" state).
//
// Suggestions the page records (`recordMyAgentSuggestion`,
// `settleMyAgentSuggestion`) are kept in `state.agentSuggestions`, and AI
// analytics counts them with the seeded ones.

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

const SUMMARIES = {
  shipment_eta: "Projected arrival for each open stop.",
  draft_reply: "Writes the email for you to review and send.",
  get_shipment: "Stops, status, equipment and references.",
  file_document: "PODs, bills of lading and lumper receipts.",
  waive_detention: "Approve or waive detention on an invoice.",
};

function matchTools(names) {
  return names.slice(0, 3).map((name) => ({ name, summary: SUMMARIES[name] ?? `${TOOLS[name] ?? ""}.` }));
}

function outcome(kind, agent, tools) {
  return { myAgentMatch: { outcome: kind, agent, tools: matchTools(tools) } };
}

export function matchAgent(state, variables, agents) {
  if (state.scenario.agentMatch === "hold") return HOLD;
  const text = String(variables.input?.text ?? "").trim();
  if (state.scenario.agentMatch === "none" || text === "") return outcome("NoMatch", null, []);

  const wide = rankTools(text, Object.keys(TOOLS));
  if (wide.length === 0) return outcome("NoMatch", null, []);
  const held = new Set(agents.flatMap((agent) => agent.toolNames ?? []));
  if (state.scenario.agentMatch === "nobody" || !wide.some((name) => held.has(name))) {
    return outcome("NoAgent", null, wide.filter((name, index) => index === 0 || name === wide[0]));
  }

  const currentId = variables.input?.currentAgentId ?? "";
  const ranked = rankTools(text, [...held]);
  const top = ranked[0];
  const candidates = agents
    .map((agent) => {
      const tools = ranked.filter((name) => (agent.toolNames ?? []).includes(name));
      const score = tools.reduce((sum, name) => sum + 1 / (ranked.indexOf(name) + 1), 0);
      return { agent, tools, score, current: agent.id === currentId };
    })
    .filter((candidate) => candidate.tools.includes(top));
  const kept = candidates.find((candidate) => candidate.current);
  if (kept) return outcome("CurrentAgent", kept.agent, kept.tools);
  candidates.sort(
    (a, b) =>
      b.score - a.score ||
      (a.agent.toolNames?.length ?? 0) - (b.agent.toolNames?.length ?? 0) ||
      a.agent.name.localeCompare(b.agent.name),
  );
  const best = candidates[0];
  if (!best) return outcome("NoMatch", null, []);

  return outcome("BetterAgent", best.agent, best.tools);
}

export function recordAgentSuggestion(state, variables) {
  const input = variables.input ?? {};
  state.agentSuggestions ??= [];
  const row = {
    id: `agsg_${String(state.agentSuggestions.length + 1).padStart(26, "0")}`,
    suggestedAgentId: input.suggestedAgentId,
    currentAgentId: input.currentAgentId ?? null,
    surface: input.surface,
    outcome: input.outcome,
    createdAt: state.now(),
    updatedAt: state.now(),
  };
  state.agentSuggestions.push(row);

  return { recordMyAgentSuggestion: { id: row.id, outcome: row.outcome } };
}

export function settleAgentSuggestion(state, variables) {
  const input = variables.input ?? {};
  const row = (state.agentSuggestions ?? []).find((suggestion) => suggestion.id === input.id);
  if (!row) throw new Error("Agent suggestion not found within your organization");
  if (row.outcome === "Shown") {
    row.outcome = input.outcome;
    row.updatedAt = state.now();
  }

  return { settleMyAgentSuggestion: { id: row.id, outcome: row.outcome } };
}
