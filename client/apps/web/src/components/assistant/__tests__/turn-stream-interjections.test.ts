import { parseAssistantStreamEvent, type AssistantStreamEvent } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { initialTurnState, reduceTurn, steeredIds, type TurnState } from "../turn-stream";

function run(events: AssistantStreamEvent[], from: TurnState = initialTurnState("Book S1")) {
  return events.reduce(reduceTurn, from);
}

/**
 * The frames as the Go runtime writes them: AssistantSteeredEvent,
 * AssistantWorldChangedEvent over conversation.WorldChange, and
 * AssistantNextTurnEvent. Optional members are left out where the server
 * omits them (omitempty), so the parse is checked against what is actually
 * sent, not against a fixture that happens to carry every field.
 */
const steeredFrame = JSON.stringify({ id: "aqm_1", content: "Use Werner instead." });
const steeredWithRecords = JSON.stringify({
  id: "aqm_2",
  content: "And move it to Tuesday.",
  mentions: [{ type: "shipment", id: "shp_1", label: "S1" }],
});
const worldFrame = JSON.stringify({
  changes: [
    {
      recordId: "shp_1",
      resource: "audited:shipment",
      label: "shipment S1",
      action: "updated",
      fields: ["status"],
      actorType: "session_user",
      actorUserId: "usr_2",
      at: 1_800_000_000,
    },
    { recordId: "shp_2", resource: "shipments", action: "deleted", at: 1_800_000_001 },
  ],
});
const nextFrame = JSON.stringify({
  turnId: "atrn_2",
  threadId: "athr_1",
  queuedId: "aqm_3",
  input: "Then bill it.",
});

describe("parsing what reaches a turn while it works", () => {
  it("reads a steer with and without the records it named", () => {
    expect(parseAssistantStreamEvent("steered", steeredFrame)).toEqual({
      event: "steered",
      data: { id: "aqm_1", content: "Use Werner instead.", mentions: [] },
    });
    const withRecords = parseAssistantStreamEvent("steered", steeredWithRecords);
    expect(withRecords?.event === "steered" && withRecords.data.mentions).toEqual([
      { type: "shipment", id: "shp_1", label: "S1" },
    ]);
  });

  it("reads changed records, filling what the server left out", () => {
    const parsed = parseAssistantStreamEvent("world_changed", worldFrame);
    expect(parsed?.event).toBe("world_changed");
    if (parsed?.event !== "world_changed") return;
    expect(parsed.data.changes[0]).toMatchObject({ label: "shipment S1", fields: ["status"] });
    expect(parsed.data.changes[1]).toMatchObject({
      recordId: "shp_2",
      label: "",
      fields: [],
      actorType: "",
    });
  });

  it("reads the turn the queue started next", () => {
    expect(parseAssistantStreamEvent("next_turn", nextFrame)).toEqual({
      event: "next_turn",
      data: { turnId: "atrn_2", threadId: "athr_1", queuedId: "aqm_3", input: "Then bill it." },
    });
  });

  it("refuses a steer with no id, which the queue could never clear", () => {
    expect(() => parseAssistantStreamEvent("steered", JSON.stringify({ content: "x" }))).toThrow();
  });
});

describe("reduceTurn with what reaches a turn while it works", () => {
  const steered = parseAssistantStreamEvent("steered", steeredFrame) as AssistantStreamEvent;
  const world = parseAssistantStreamEvent("world_changed", worldFrame) as AssistantStreamEvent;

  it("keeps steers and changes in the order they arrived", () => {
    const state = run([steered, world]);

    expect(state.interjections.map((item) => item.kind)).toEqual(["steer", "world"]);
    expect(state.interjections[0]).toEqual({
      kind: "steer",
      id: "aqm_1",
      text: "Use Werner instead.",
      mentions: [],
    });
    expect(steeredIds(state.interjections)).toEqual(new Set(["aqm_1"]));
  });

  it("does not repeat them for a reader that rejoins from the start", () => {
    const state = run([steered, world, steered, world]);

    expect(state.interjections).toHaveLength(2);
  });

  it("ignores a change notice that names no record", () => {
    const empty = parseAssistantStreamEvent("world_changed", JSON.stringify({ changes: null }));

    expect(run([empty as AssistantStreamEvent]).interjections).toEqual([]);
  });

  it("remembers the turn the queue started next", () => {
    const next = parseAssistantStreamEvent("next_turn", nextFrame) as AssistantStreamEvent;

    expect(run([next]).next).toEqual({
      turnId: "atrn_2",
      queuedId: "aqm_3",
      input: "Then bill it.",
    });
  });
});
