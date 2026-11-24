import { describe, expect, it } from "vitest";
import { heldSettlementActions, settlementActions } from "../../../record-card/settlement-values";
import { settlementFlags } from "../driver-settlement-card";
import { DAY, HOUR, pendingSettlement, t } from "./fixtures";

const held = pendingSettlement({
  holdReason: "RateDispute",
  holdNote: "Waiting on the broker's corrected rate confirmation",
  heldAt: DAY.oct13 + 18 * HOUR,
  heldBy: { id: "usr_mia", name: "Mia Chen" },
});

describe("a held settlement on the Desk card", () => {
  it("leads its flags with the hold, its reason in words and who placed it", () => {
    const flags = settlementFlags(held, t);

    expect(flags[0]).toEqual({
      key: "hold",
      severity: "critical",
      title: "On hold: Rate dispute",
      body:
        "Waiting on the broker's corrected rate confirmation Held by Mia Chen. Approving and " +
        "posting wait until the hold is released.",
    });
    expect(flags.slice(1).map((flag) => flag.key)).toEqual(["ManualAdjustment", "HighVariance"]);
  });

  it("raises no hold flag on a settlement that is not held", () => {
    expect(settlementFlags(pendingSettlement(), t).map((flag) => flag.key)).not.toContain("hold");
  });

  it("does not offer approving or posting while it is held", () => {
    expect(heldSettlementActions(settlementActions("PendingApproval"))).toEqual({
      primary: null,
      more: ["reject", "void"],
    });
    expect(heldSettlementActions(settlementActions("Approved"))).toEqual({
      primary: null,
      more: ["void"],
    });
    expect(heldSettlementActions(settlementActions("Draft"))).toEqual(settlementActions("Draft"));
  });
});
