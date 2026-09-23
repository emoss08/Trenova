import type { AgentTemplateKind, AssistantPageContext } from "@/types/assistant";

export type Suggestion = {
  label: string;
  prompt: string;
};

/**
 * Opening questions that show what an agent is for, so the first thing a
 * person sees is not an empty box. Each one is answerable with the tools that
 * template can hold; a suggestion the agent would refuse teaches the wrong
 * lesson on the first click.
 */
const SUGGESTIONS: Partial<Record<AgentTemplateKind, Suggestion[]>> = {
  DispatchAssistant: [
    {
      label: "Where is a shipment right now?",
      prompt: "Where is PRO S12345 right now and who is on it?",
    },
    {
      label: "What is picking up today?",
      prompt: "Which shipments are scheduled to pick up today?",
    },
    {
      label: "Is a driver available?",
      prompt: "Is Maria Ortiz available for a load tomorrow morning?",
    },
  ],
  BillingAssistant: [
    {
      label: "What is blocking an invoice?",
      prompt: "What is holding PRO S12345 in the billing queue?",
    },
    {
      label: "Oldest blocked items",
      prompt: "Which billing queue items have been blocked the longest?",
    },
    {
      label: "Missing documents",
      prompt: "Which delivered shipments are still missing a signed bill of lading?",
    },
  ],
  ComplianceAssistant: [
    {
      label: "Expiring credentials",
      prompt: "Which drivers have a medical card expiring in the next 30 days?",
    },
    { label: "Check a driver", prompt: "Is Maria Ortiz qualified to drive today?" },
    { label: "Hazmat endorsements", prompt: "Which drivers hold a current hazmat endorsement?" },
  ],
  CustomerAssistant: [
    { label: "Shipment status", prompt: "What is the status of PRO S12345?" },
    { label: "Delivery estimate", prompt: "When will PRO S12345 be delivered?" },
    { label: "Recent shipments", prompt: "What shipped for Acme Manufacturing this week?" },
  ],
  GeneralAssistant: [
    { label: "Create a rate matrix", prompt: "How do I create a rate matrix for a customer?" },
    {
      label: "Approve a proposal",
      prompt: "Where do I approve a change the billing agent proposed?",
    },
    { label: "Add a driver", prompt: "How do I add a new driver and their credentials?" },
  ],
};

const DEFAULT_SUGGESTIONS: Suggestion[] = [
  { label: "What can you do?", prompt: "What can you look up or change for me?" },
  { label: "Where is a shipment?", prompt: "Where is PRO S12345 right now?" },
  { label: "What needs attention?", prompt: "What needs my attention today?" },
];

/**
 * Questions about the page itself, ahead of the agent's own: a filtered
 * table, an open record, figures on screen. Each is answerable from the
 * page context the message carries, so the first click on a busy page asks
 * about what is in front of the person.
 */
export function pageSuggestions(page: AssistantPageContext | null | undefined): Suggestion[] {
  if (!page) {
    return [];
  }
  const suggestions: Suggestion[] = [];
  const view = page.view ?? null;
  const filtered =
    view !== null &&
    ((view.fieldFilters?.length ?? 0) > 0 ||
      (view.filterGroups?.length ?? 0) > 0 ||
      (view.query ?? "").trim() !== "");
  if (filtered) {
    suggestions.push({
      label: "Explain these filters",
      prompt: "Explain what this table is filtered to and what the rows have in common.",
    });
  }
  if (page.entityType !== "" && page.entityId !== "") {
    const kind = page.entityType.replaceAll("_", " ");
    suggestions.push(
      {
        label: `Why is this ${kind} flagged?`,
        prompt: `Look at the ${kind} I have open and tell me whether anything about it needs attention, and why.`,
      },
      {
        label: `Summarize this ${kind}`,
        prompt: `Summarize the ${kind} I have open: what it is, where it stands, and what happens next.`,
      },
    );
  }
  if ((view?.kpis?.length ?? 0) > 0) {
    suggestions.push({
      label: "What stands out in these figures?",
      prompt: "Look at the figures on this page and tell me which ones stand out and why.",
    });
  }
  if ((view?.selection?.count ?? 0) > 0) {
    suggestions.push({
      label: "What do the selected rows have in common?",
      prompt: "Look at the rows I have selected and tell me what they have in common.",
    });
  }

  return suggestions;
}

export function suggestionsFor(
  template: AgentTemplateKind | null | undefined,
  page?: AssistantPageContext | null,
): Suggestion[] {
  const own = (template && SUGGESTIONS[template]) || DEFAULT_SUGGESTIONS;

  return [...pageSuggestions(page), ...own];
}

type SuggestionSource = {
  template?: AgentTemplateKind | null;
  /** The agent's own opening questions, from the server. */
  starters?: readonly Suggestion[] | null;
};

/**
 * The opening questions for one agent: the page's own first, then the
 * agent's. The agent's come from the server, which draws them from its
 * template or, for an agent built by hand, from the tools it holds, so the
 * Report Builder is never offered "Where is a shipment?". The template table
 * above only answers for a definition read before the server sent starters.
 */
export function agentSuggestions(
  agent: SuggestionSource | null | undefined,
  page?: AssistantPageContext | null,
): Suggestion[] {
  if (!agent) {
    return pageSuggestions(page);
  }
  const own =
    agent.starters && agent.starters.length > 0
      ? agent.starters
      : (agent.template && SUGGESTIONS[agent.template]) || DEFAULT_SUGGESTIONS;

  return [...pageSuggestions(page), ...own];
}
