import type { AgentTemplateKind } from "@/types/assistant";

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

export function suggestionsFor(template: AgentTemplateKind | null | undefined): Suggestion[] {
  return (template && SUGGESTIONS[template]) || DEFAULT_SUGGESTIONS;
}
