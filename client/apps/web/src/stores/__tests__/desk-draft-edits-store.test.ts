import { beforeEach, describe, expect, it } from "vitest";
import { useDraftEditsStore } from "../desk-draft-edits-store";

describe("draft edits sent for approval", () => {
  beforeEach(() => useDraftEditsStore.setState({ edits: {} }));

  it("keeps the changed wording for the decision card to approve with", () => {
    useDraftEditsStore.getState().setEdits("prop_1", { subject: "PO needed" });
    expect(useDraftEditsStore.getState().edits.prop_1).toEqual({ subject: "PO needed" });
  });

  it("keeps nothing when nothing changed, so the card approves as proposed", () => {
    useDraftEditsStore.getState().setEdits("prop_1", { subject: "PO needed" });
    useDraftEditsStore.getState().setEdits("prop_1", {});
    expect(useDraftEditsStore.getState().edits).toEqual({});
  });

  it("forgets the wording once the proposal is decided", () => {
    useDraftEditsStore.getState().setEdits("prop_1", { body: "Could you send it over?" });
    useDraftEditsStore.getState().clearEdits("prop_1");
    expect(useDraftEditsStore.getState().edits.prop_1).toBeUndefined();
  });
});
