import { DESK_DECISIONS_PATH, deskDecisionPath } from "@/lib/desk-paths";
import { describe, expect, it } from "vitest";

describe("the Desk's decisions address", () => {
  it("is the Decisions lens of Needs you", () => {
    expect(DESK_DECISIONS_PATH).toBe("/desk/needs?lens=decisions");
  });

  it("opens on one change or on the change a run waits on", () => {
    expect(deskDecisionPath({ proposal: "aprop_1" })).toBe(
      "/desk/needs?lens=decisions&proposal=aprop_1",
    );
    expect(deskDecisionPath({ run: "arun 1&x" })).toBe(
      "/desk/needs?lens=decisions&run=arun%201%26x",
    );
  });
});
