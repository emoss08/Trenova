import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { agentSuggestions, suggestionsFor } from "../suggestions";

/**
 * The starter questions follow the page. On a filtered table the first
 * question is about those rows; with a record open, about that record; with
 * figures on screen, about the figures. Away from any of that, the agent's
 * own questions stand.
 */
describe("suggestionsFor with a page", () => {
  it("asks about the filtered rows when the page is a filtered table", () => {
    const suggestions = suggestionsFor("DispatchAssistant", {
      path: "/shipments",
      entityType: "shipment",
      entityId: "",
      title: "Shipments",
      view: {
        resource: "shipment",
        fieldFilters: [{ field: "status", operator: "eq", value: "Delayed" }],
      },
    });

    expect(suggestions[0].label).toBe("Explain these filters");
    expect(suggestions[0].prompt).toContain("filtered");
    expect(suggestions.some((item) => item.label === "Where is a shipment right now?")).toBe(true);
  });

  it("asks about the open record when one is open", () => {
    const suggestions = suggestionsFor("DispatchAssistant", {
      path: "/shipments?panelEntityId=shp_1",
      entityType: "shipment",
      entityId: "shp_1",
      title: "Shipments",
    });

    expect(suggestions[0].label).toBe("Why is this record flagged?");
    expect(suggestions[1].label).toBe("Summarize this record");
  });

  it("asks about the figures when the page shows some", () => {
    const suggestions = suggestionsFor("BillingAssistant", {
      path: "/billing",
      entityType: "",
      entityId: "",
      title: "Billing",
      view: { resource: "invoice", kpis: [{ label: "Unbilled", value: "$12,400" }] },
    });

    expect(suggestions[0].label).toBe("What stands out in these figures?");
  });

  it("keeps the agent's own questions away from any page", () => {
    expect(suggestionsFor("DispatchAssistant", null)[0].label).toBe(
      "Where is a shipment right now?",
    );
    expect(
      suggestionsFor("DispatchAssistant", { path: "/", entityType: "", entityId: "", title: "" })[0]
        .label,
    ).toBe("Where is a shipment right now?");
  });
});

/**
 * An agent's own starters come from the server and win over the template
 * table, so an agent built by hand is offered questions it can answer.
 */
describe("agentSuggestions", () => {
  it("uses the starters the server sent", () => {
    const suggestions = agentSuggestions({
      template: null,
      starters: [{ label: "Build a report", prompt: "Build a report of in-transit shipments." }],
    });

    expect(suggestions).toEqual([
      { label: "Build a report", prompt: "Build a report of in-transit shipments." },
    ]);
  });

  it("falls back to the template's questions when there are no starters", () => {
    const suggestions = agentSuggestions({ template: "BillingAssistant", starters: [] });

    expect(suggestions[0].label).toBe("What is blocking an invoice?");
  });

  // Each money assistant opens on its own work rather than the generic
  // "what can you do?", which is what an agent with no table entry gets.
  it("opens the settlements clerk and receivables on their own work", () => {
    const clerk = agentSuggestions({ template: "SettlementsClerk", starters: [] });
    const receivables = agentSuggestions({ template: "Receivables", starters: [] });

    expect(clerk.map((item) => item.label)).toContain("Settlements with exceptions");
    expect(receivables.map((item) => item.label)).toContain("Who to chase today");
    expect(clerk.some((item) => item.label === "What can you do?")).toBe(false);
    expect(receivables.some((item) => item.label === "What can you do?")).toBe(false);
  });

  it("opens the focused clerks on their own work", () => {
    const steward = agentSuggestions({ template: "MasterDataSteward", starters: [] });
    const workforce = agentSuggestions({ template: "WorkforceCoordinator", starters: [] });
    const fuel = agentSuggestions({ template: "FuelTaxClerk", starters: [] });
    const reports = agentSuggestions({ template: "ReportAnalyst", starters: [] });

    expect(steward.map((item) => item.label)).toContain("Paperwork to file");
    expect(workforce.map((item) => item.label)).toContain("Time off to decide");
    expect(fuel.map((item) => item.label)).toContain("Is the IFTA return ready?");
    expect(reports.map((item) => item.label)).toContain("Which report answers this?");
    for (const suggestions of [steward, workforce, fuel, reports]) {
      expect(suggestions.some((item) => item.label === "What can you do?")).toBe(false);
    }
  });

  it("puts the page's questions ahead of the agent's", () => {
    const suggestions = agentSuggestions(
      { template: null, starters: [{ label: "Own", prompt: "Own question" }] },
      { path: "/shipments", entityType: "shipment", entityId: "shp_1", title: "S1" },
    );

    expect(suggestions[0].label).toBe("Why is this record flagged?");
    expect(suggestions.at(-1)?.label).toBe("Own");
  });

  it("offers only the page's questions when there is no agent", () => {
    expect(agentSuggestions(null)).toEqual([]);
  });
});

/**
 * A suggestion is shown as a chip and sent as the person's own message, so a person reading
 * Spanish is offered, and sends, a Spanish question.
 */
describe("suggestions in another language", () => {
  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({
        "What can you do?": "¿Qué puede hacer?",
        "What can you look up or change for me?": "¿Qué puede consultar o cambiar por mí?",
      }),
    });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("offers and sends the question in the language on screen", () => {
    const [first] = suggestionsFor(null, null);

    expect(first.label).toBe("¿Qué puede hacer?");
    expect(first.prompt).toBe("¿Qué puede consultar o cambiar por mí?");
  });
});
