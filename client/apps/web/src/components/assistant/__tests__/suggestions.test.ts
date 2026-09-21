import { describe, expect, it } from "vitest";
import { suggestionsFor } from "../suggestions";

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

    expect(suggestions[0].label).toBe("Why is this shipment flagged?");
    expect(suggestions[1].label).toBe("Summarize this shipment");
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
