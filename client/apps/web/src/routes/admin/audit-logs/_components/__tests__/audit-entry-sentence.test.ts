import { describe, expect, it } from "vitest";
import {
  actorKindOfChannel,
  actorKindOfPrincipal,
  changeChips,
  entrySentence,
} from "../audit-entry-sentence";

describe("entrySentence", () => {
  it("names the record by its reference when the entry carries one", () => {
    expect(
      entrySentence({ operation: "update", resource: "shipment", entityRef: "PRO 1048217" }),
    ).toBe("Updated PRO 1048217");
  });

  it("falls back to the kind of record when there is no reference", () => {
    expect(
      entrySentence({ operation: "delete", resource: "accessorial_rate", entityRef: null }),
    ).toBe("Deleted Accessorial Rate");
  });

  it("reads the operation whatever its case", () => {
    expect(
      entrySentence({ operation: "EXPORT", resource: "shipment", entityRef: "2,310 records" }),
    ).toBe("Exported 2,310 records");
  });

  it("names an operation it has no sentence for beside its record", () => {
    expect(entrySentence({ operation: "rebuild", resource: "shipment", entityRef: "PRO 1" })).toBe(
      "rebuild · PRO 1",
    );
  });
});

describe("changeChips", () => {
  it("lists the first fields sorted by path with before and after, and counts the rest", () => {
    const { chips, more } = changeChips(
      {
        status: { from: "Posted", to: "Void" },
        balance: { from: "2418.60", to: "0.00" },
        reason: { to: "Duplicate" },
        notes: { from: "old" },
      },
      3,
    );

    expect(chips).toEqual([
      { path: "balance", field: "Balance", from: "2418.60", to: "0.00" },
      { path: "notes", field: "Notes", from: "old", to: null },
      { path: "reason", field: "Reason", from: null, to: "Duplicate" },
    ]);
    expect(more).toBe(1);
  });

  it("shortens a long value so a chip stays one line", () => {
    const long = "x".repeat(60);
    const { chips } = changeChips({ memo: { from: "a", to: long } }, 3);

    expect(chips[0].to).toBe(`${"x".repeat(39)}…`);
  });

  it("has nothing to show when the entry recorded no changes", () => {
    expect(changeChips(null, 3)).toEqual({ chips: [], more: 0 });
    expect(changeChips(["not", "a", "record"], 3)).toEqual({ chips: [], more: 0 });
  });
});

describe("actor kinds", () => {
  it("sorts channels into the chart's four series", () => {
    expect(actorKindOfChannel("web")).toBe("people");
    expect(actorKindOfChannel("support")).toBe("people");
    expect(actorKindOfChannel("agent_conversation")).toBe("agents");
    expect(actorKindOfChannel("agent_run")).toBe("agents");
    expect(actorKindOfChannel("scim")).toBe("integrations");
    expect(actorKindOfChannel("system")).toBe("jobs");
  });

  it("sorts principals the way the server counts who acted", () => {
    expect(actorKindOfPrincipal("session_user")).toBe("people");
    expect(actorKindOfPrincipal("agent")).toBe("agents");
    expect(actorKindOfPrincipal("api_key")).toBe("integrations");
    expect(actorKindOfPrincipal("system")).toBe("jobs");
  });
});
