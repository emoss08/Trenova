import type { AgentTemplateKind, AssistantPageContext } from "@/types/assistant";
import { translate } from "@trenova/shared/i18n/runtime";

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
// Built when asked for, never at import: a suggestion is shown as a chip and sent as the
// person's own message, so both halves read in the language on screen.
function templateSuggestions(template: AgentTemplateKind): Suggestion[] | undefined {
  const byTemplate: Partial<Record<AgentTemplateKind, Suggestion[]>> = {
    DispatchAssistant: [
      {
        label: translate("Where is a shipment right now?"),
        prompt: translate("Where is PRO S12345 right now and who is on it?"),
      },
      {
        label: translate("What is picking up today?"),
        prompt: translate("Which shipments are scheduled to pick up today?"),
      },
      {
        label: translate("Is a driver available?"),
        prompt: translate("Is Maria Ortiz available for a load tomorrow morning?"),
      },
    ],
    BillingAssistant: [
      {
        label: translate("What is blocking an invoice?"),
        prompt: translate("What is holding PRO S12345 in the billing queue?"),
      },
      {
        label: translate("Oldest blocked items"),
        prompt: translate("Which billing queue items have been blocked the longest?"),
      },
      {
        label: translate("Missing documents"),
        prompt: translate("Which delivered shipments are still missing a signed bill of lading?"),
      },
    ],
    ComplianceAssistant: [
      {
        label: translate("Expiring credentials"),
        prompt: translate("Which drivers have a medical card expiring in the next 30 days?"),
      },
      {
        label: translate("Check a driver"),
        prompt: translate("Is Maria Ortiz qualified to drive today?"),
      },
      {
        label: translate("Hazmat endorsements"),
        prompt: translate("Which drivers hold a current hazmat endorsement?"),
      },
    ],
    CustomerAssistant: [
      {
        label: translate("Shipment status"),
        prompt: translate("What is the status of PRO S12345?"),
      },
      {
        label: translate("Delivery estimate"),
        prompt: translate("When will PRO S12345 be delivered?"),
      },
      {
        label: translate("Recent shipments"),
        prompt: translate("What shipped for Acme Manufacturing this week?"),
      },
    ],
    SettlementsClerk: [
      {
        label: translate("Why is a load missing?"),
        prompt: translate("Why is PRO S12345 missing from Maria Ortiz's settlement?"),
      },
      {
        label: translate("Settlements with exceptions"),
        prompt: translate("Which driver settlements this period have exceptions or open disputes?"),
      },
      {
        label: translate("Carrier invoice variances"),
        prompt: translate("Which carrier invoices differ from what the load was expected to cost?"),
      },
    ],
    Receivables: [
      {
        label: translate("Who to chase today"),
        prompt: translate("Which overdue invoices should I chase today, most urgent first?"),
      },
      {
        label: translate("A customer's balance"),
        prompt: translate("What does Acme Manufacturing owe us, and how late is it?"),
      },
      {
        label: translate("Unapplied cash"),
        prompt: translate("Which customer payments still have cash not applied to an invoice?"),
      },
    ],
    MasterDataSteward: [
      {
        label: translate("Is a carrier on file?"),
        prompt: translate(
          "Is the carrier with DOT number 1234567 already on file, and is it active?",
        ),
      },
      {
        label: translate("Paperwork to file"),
        prompt: translate(
          "Which scanned documents are waiting to be filed, and where does each belong?",
        ),
      },
      {
        label: translate("Open watchtower items"),
        prompt: translate("Which watchtower items are open, and which are already dealt with?"),
      },
    ],
    WorkforceCoordinator: [
      {
        label: translate("Time off to decide"),
        prompt: translate(
          "Which time-off requests are waiting for a decision, and what is each driver covering?",
        ),
      },
      {
        label: translate("Open leave cases"),
        prompt: translate("Which leave cases are open, and what does each one still need?"),
      },
      {
        label: translate("Random testing round"),
        prompt: translate(
          "Draw this period's random drug and alcohol testing round from our pool.",
        ),
      },
    ],
    FuelTaxClerk: [
      {
        label: translate("Is the IFTA return ready?"),
        prompt: translate(
          "Is last quarter's IFTA return drafted, and what still blocks finalizing it?",
        ),
      },
      {
        label: translate("Held fuel imports"),
        prompt: translate("Which fuel card statements have rows held back, and why?"),
      },
      {
        label: translate("Unassigned fuel cards"),
        prompt: translate("Which fuel cards are not assigned to a tractor or a driver?"),
      },
    ],
    ReportAnalyst: [
      {
        label: translate("Which report answers this?"),
        prompt: translate(
          "Which of our reports shows on-time delivery by customer, and what does it contain?",
        ),
      },
      {
        label: translate("What moved since last week?"),
        prompt: translate(
          "Run the lane profitability report again and tell me what moved since last week.",
        ),
      },
      {
        label: translate("Email a report weekly"),
        prompt: translate(
          "Email the late shipments report to the operations team every Monday morning.",
        ),
      },
    ],
    GeneralAssistant: [
      {
        label: translate("Create a rate matrix"),
        prompt: translate("How do I create a rate matrix for a customer?"),
      },
      {
        label: translate("Approve a proposal"),
        prompt: translate("Where do I approve a change the billing agent proposed?"),
      },
      {
        label: translate("Add a driver"),
        prompt: translate("How do I add a new driver and their credentials?"),
      },
    ],
  };
  return byTemplate[template];
}

function defaultSuggestions(): Suggestion[] {
  return [
    {
      label: translate("What can you do?"),
      prompt: translate("What can you look up or change for me?"),
    },
    {
      label: translate("Where is a shipment?"),
      prompt: translate("Where is PRO S12345 right now?"),
    },
    {
      label: translate("What needs attention?"),
      prompt: translate("What needs my attention today?"),
    },
  ];
}

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
      label: translate("Explain these filters"),
      prompt: translate("Explain what this table is filtered to and what the rows have in common."),
    });
  }
  if (page.entityType !== "" && page.entityId !== "") {
    suggestions.push(
      {
        label: translate("Why is this record flagged?"),
        prompt: translate(
          "Look at the record I have open and tell me whether anything about it needs attention, and why.",
        ),
      },
      {
        label: translate("Summarize this record"),
        prompt: translate(
          "Summarize the record I have open: what it is, where it stands, and what happens next.",
        ),
      },
    );
  }
  if ((view?.kpis?.length ?? 0) > 0) {
    suggestions.push({
      label: translate("What stands out in these figures?"),
      prompt: translate(
        "Look at the figures on this page and tell me which ones stand out and why.",
      ),
    });
  }
  if ((view?.selection?.count ?? 0) > 0) {
    suggestions.push({
      label: translate("What do the selected rows have in common?"),
      prompt: translate("Look at the rows I have selected and tell me what they have in common."),
    });
  }

  return suggestions;
}

export function suggestionsFor(
  template: AgentTemplateKind | null | undefined,
  page?: AssistantPageContext | null,
): Suggestion[] {
  const own = (template && templateSuggestions(template)) || defaultSuggestions();

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
      : (agent.template && templateSuggestions(agent.template)) || defaultSuggestions();

  return [...pageSuggestions(page), ...own];
}
