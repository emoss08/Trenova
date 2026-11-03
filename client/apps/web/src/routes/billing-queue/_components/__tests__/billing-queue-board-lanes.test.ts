import { describe, expect, it } from "vitest";
import {
  laneDropId,
  laneOf,
  lanesAfterCross,
  lanesAfterDrop,
  lanesWithGroup,
  lanesWithoutGroup,
  sameOrder,
} from "../billing-queue-board-lanes";

const lanes = { packet: ["pod", "bol"], out: ["lumper", "scale"] };

describe("packet board lanes", () => {
  it("names the lane of a card or of a lane's own drop target", () => {
    expect(laneOf(lanes, "bol")).toBe("packet");
    expect(laneOf(lanes, "scale")).toBe("out");
    expect(laneOf(lanes, laneDropId("out"))).toBe("out");
    expect(laneOf(lanes, "unknown")).toBeNull();
  });

  it("moves a card into the other lane before the card it is over", () => {
    expect(lanesAfterCross(lanes, "lumper", "bol", false)).toEqual({
      packet: ["pod", "lumper", "bol"],
      out: ["scale"],
    });
  });

  it("moves a card in after the card it is over once the pointer is past that card's middle", () => {
    expect(lanesAfterCross(lanes, "lumper", "bol", true)).toEqual({
      packet: ["pod", "bol", "lumper"],
      out: ["scale"],
    });
  });

  it("puts a card dragged over the empty part of a lane last", () => {
    expect(lanesAfterCross(lanes, "pod", laneDropId("out"), false)).toEqual({
      packet: ["bol"],
      out: ["lumper", "scale", "pod"],
    });
  });

  it("leaves a card over its own lane to the sortable", () => {
    expect(lanesAfterCross(lanes, "pod", "bol", true)).toBe(lanes);
  });

  it("moves a card dropped on another card in its lane to that card's place", () => {
    expect(lanesAfterDrop(lanes, "bol", "pod")).toEqual({
      packet: ["bol", "pod"],
      out: ["lumper", "scale"],
    });
    expect(lanesAfterDrop(lanes, "lumper", "scale")).toEqual({
      packet: ["pod", "bol"],
      out: ["scale", "lumper"],
    });
  });

  it("keeps the lanes as they are for a drop on nothing, on itself or on a lane", () => {
    expect(lanesAfterDrop(lanes, "bol", null)).toBe(lanes);
    expect(lanesAfterDrop(lanes, "bol", "bol")).toBe(lanes);
    expect(lanesAfterDrop(lanes, "bol", laneDropId("packet"))).toBe(lanes);
  });

  it("tells a changed order from the same one", () => {
    expect(sameOrder(["pod", "bol"], ["pod", "bol"])).toBe(true);
    expect(sameOrder(["bol", "pod"], ["pod", "bol"])).toBe(false);
    expect(sameOrder(["pod"], ["pod", "bol"])).toBe(false);
  });

  it("folds the rest of a dragged group into the card in hand", () => {
    expect(lanesWithoutGroup(lanes, "bol", ["pod", "bol", "scale"])).toEqual({
      packet: ["bol"],
      out: ["lumper"],
    });
  });

  it("lands the rest of a group after the card in hand, in the group's order", () => {
    const folded = { packet: ["bol"], out: ["lumper"] };

    expect(lanesWithGroup(folded, "bol", ["pod", "bol", "scale"])).toEqual({
      packet: ["bol", "pod", "scale"],
      out: ["lumper"],
    });
  });

  it("lands a group dropped out of the packet beside the card in hand there", () => {
    const dropped = { packet: [], out: ["lumper", "bol"] };

    expect(lanesWithGroup(dropped, "bol", ["pod", "bol"])).toEqual({
      packet: [],
      out: ["lumper", "bol", "pod"],
    });
  });

  it("leaves a single card's lanes alone", () => {
    expect(lanesWithoutGroup(lanes, "pod", ["pod"])).toBe(lanes);
    expect(lanesWithGroup(lanes, "pod", ["pod"])).toBe(lanes);
  });
});
