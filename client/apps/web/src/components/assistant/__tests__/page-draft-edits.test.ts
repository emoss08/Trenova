import {
  assistantArtifactEventSchema,
  pageContextSchema,
  pageThreadSchema,
  type AssistantArtifactEvent,
} from "@/types/assistant";
import { afterEach, describe, expect, it } from "vitest";
import { claimDraftEdit, nextDraftEdits } from "../page-draft-edits";

/*
 * Fixtures follow the server: an artifact event is
 * services.AssistantArtifactEvent, whose `draft` is a pagedraft.Edit set only
 * on draft_edit artifacts; a page thread is services.PageThread.
 */
function edit(overrides: Record<string, unknown> = {}) {
  return {
    surface: "shipment_import",
    action: "accept_field",
    fieldKey: "loadNumber",
    label: "Load Number",
    ...overrides,
  };
}

function event(overrides: Record<string, unknown> = {}): AssistantArtifactEvent {
  return assistantArtifactEventSchema.parse({
    id: "aart_1",
    kind: "draft_edit",
    status: "Ready",
    title: "Accepted Load Number",
    sourceToolCallId: "call_1",
    draft: edit(),
    ...overrides,
  });
}

function claims() {
  const seen = new Set<string>();
  return (id: string) => {
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  };
}

afterEach(() => {
  sessionStorage.clear();
});

describe("nextDraftEdits", () => {
  it("hands the page each change the assistant made, in the order it made them", () => {
    const announced = [
      event({ id: "aart_1", draft: edit({ fieldKey: "loadNumber" }) }),
      event({
        id: "aart_2",
        draft: edit({ action: "set_stop_location", fieldKey: "", stopIndex: 1, value: "loc_1" }),
      }),
    ];

    const edits = nextDraftEdits(announced, "shipment_import", claims());

    expect(edits.map((item) => item.action)).toEqual(["accept_field", "set_stop_location"]);
    expect(edits[1].stopIndex).toBe(1);
    expect(edits[1].value).toBe("loc_1");
  });

  it("applies a change once, however often the turn is replayed", () => {
    const claim = claims();
    const announced = [event({})];

    expect(nextDraftEdits(announced, "shipment_import", claim)).toHaveLength(1);
    expect(nextDraftEdits(announced, "shipment_import", claim)).toEqual([]);
    expect(
      nextDraftEdits([...announced, event({ id: "aart_2" })], "shipment_import", claim),
    ).toHaveLength(1);
  });

  it("stops at stop 0, which is a real stop and not an absent index", () => {
    const [applied] = nextDraftEdits(
      [event({ draft: edit({ action: "set_stop_location", stopIndex: 0, value: "loc_9" }) })],
      "shipment_import",
      claims(),
    );

    expect(applied.stopIndex).toBe(0);
  });

  it("ignores other artifacts, other pages' changes, and a change it cannot read", () => {
    const claim = claims();
    const announced = [
      event({ id: "aart_nav", kind: "navigation", path: "/billing", draft: null }),
      event({
        id: "aart_formula",
        draft: {
          surface: "formula",
          action: "propose_formula",
          formula: {
            schemaId: "shipment",
            expression: "1",
            variables: null,
            check: { valid: true },
            scenarios: null,
          },
        },
      }),
      event({ id: "aart_bad", draft: { surface: "shipment_import", action: "delete_everything" } }),
    ];

    expect(nextDraftEdits(announced, "shipment_import", claim)).toEqual([]);
    // Only the page's own changes are claimed; the formula one is left for its page.
    expect(nextDraftEdits(announced, "formula", claim).map((item) => item.action)).toEqual([
      "propose_formula",
    ]);
  });

  it("claims through the tab's session, so a second reader of the turn does not reapply", () => {
    expect(claimDraftEdit("aart_session")).toBe(true);
    expect(claimDraftEdit("aart_session")).toBe(false);
  });
});

describe("assistantArtifactEventSchema", () => {
  it("keeps an artifact whose draft it cannot read, without the draft", () => {
    const parsed = event({ id: "aart_bad", draft: { surface: "nowhere" } });

    expect(parsed.id).toBe("aart_bad");
    expect(parsed.draft).toBeNull();
  });

  it("reads a formula proposal whose lists the server left null", () => {
    const parsed = event({
      draft: {
        surface: "formula",
        action: "propose_formula",
        formula: {
          schemaId: "shipment",
          expression: "totalDistance * 2",
          variables: null,
          explanation: "Two dollars a mile.",
          check: { valid: true, result: "200" },
          scenarios: null,
        },
      },
    });

    expect(parsed.draft?.formula?.variables).toEqual([]);
    expect(parsed.draft?.formula?.scenarios).toEqual([]);
    expect(parsed.draft?.formula?.check.error).toBe("");
  });
});

describe("pageThreadSchema", () => {
  it("reads the conversation and the agent the page talks to", () => {
    const parsed = pageThreadSchema.parse({
      thread: {
        id: "athr_1",
        businessUnitId: "bu_1",
        organizationId: "org_1",
        userId: "usr_1",
        agentDefinitionId: "agdef_1",
        title: "Shipment import",
        status: "Active",
        origin: "Import",
        subjectType: "Document",
        subjectId: "doc_1",
        canContinue: true,
        taint: { marks: [{ source: "Document", toolName: "", callId: "", at: 10 }] },
        taintedAt: 10,
        version: 1,
        createdAt: 10,
        updatedAt: 10,
      },
      agent: {
        id: "agdef_1",
        name: "Shipment import assistant",
        description: "",
        template: "ImportAssistant",
        icon: "",
        accent: "",
        systemKey: "import_assistant",
        toolNames: null,
        starters: null,
      },
    });

    expect(parsed.thread.origin).toBe("Import");
    expect(parsed.thread.taintedAt).toBe(10);
    expect(parsed.agent.toolNames).toEqual([]);
    expect(parsed.agent.starters).toEqual([]);
  });
});

describe("pageContextSchema", () => {
  // A saved message carries the page context it was asked with. A draft this
  // build cannot read must not make the conversation's history unreadable.
  it("reads a message whose saved draft it does not understand, without the draft", () => {
    const parsed = pageContextSchema.parse({
      path: "/shipment-management/shipments/import",
      draft: { surface: "somewhere_new", somethingElse: {} },
    });

    expect(parsed.path).toBe("/shipment-management/shipments/import");
    expect(parsed.draft).toBeNull();
  });

  it("keeps a draft it can read", () => {
    const parsed = pageContextSchema.parse({
      path: "/billing/configuration-files/formula-templates/new",
      draft: {
        surface: "formula",
        formula: {
          schemaId: "shipment",
          templateType: "FreightCharge",
          expression: "1",
          variables: [],
        },
      },
    });

    expect(parsed.draft?.formula?.expression).toBe("1");
  });
});
