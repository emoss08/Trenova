import { toDeskDetention } from "@/lib/graphql/detention-card";
import { describe, expect, it } from "vitest";
import {
  clockModel,
  detentionActions,
  detentionFacts,
  detentionFigure,
  detentionKicker,
  detentionStanding,
  detentionSubtitle,
  detentionTrack,
  detentionTrackNote,
  noticeReading,
  noticeRows,
  noticeRule,
  noticeSlip,
  paySplit,
  placeLine,
  proofReading,
  receiptRows,
  settledSlip,
} from "../detention-occurrence-card";
import {
  ACCRUING_NOW,
  accruingCharge,
  at,
  pendingCharge,
  rawDetail,
  rawOccurrence,
  snapshotJson,
  t,
  waivedCharge,
} from "./fixtures";

const AFTER = at(16);

describe("toDeskDetention", () => {
  it("parses the snapshot and the trace, decimals the server sends as strings included", () => {
    const { occurrence } = pendingCharge();

    expect(occurrence.snapshot?.requireApprovalOverAmount).toBe(250);
    expect(occurrence.snapshot?.tiers?.[0]?.rate).toBe(75);
    expect(occurrence.trace?.steps.map((step) => step.kind)).toEqual([
      "PolicyResolved",
      "ClockStart",
      "ClockStop",
      "RawDwell",
      "FreeTime",
      "Rounding",
      "TierApplied",
      "NotificationGate",
      "DriverPay",
      "Final",
    ]);
  });

  it("reads null evidence and notices as empty lists", () => {
    const detail = toDeskDetention(rawDetail({ evidence: null, notices: null }));

    expect(detail.evidence).toEqual([]);
    expect(detail.notices).toEqual([]);
  });

  it("leaves a snapshot or trace it cannot read null rather than failing the card", () => {
    const detail = toDeskDetention(
      rawDetail({
        occurrence: rawOccurrence({ policySnapshot: { policyName: 7 }, calculationTrace: "x" }),
      }),
    );

    expect(detail.occurrence.snapshot).toBeNull();
    expect(detail.occurrence.trace).toBeNull();
  });
});

describe("the head", () => {
  it("names the load, the stop and the policy with its version above the facility", () => {
    const { occurrence } = pendingCharge();

    expect(detentionKicker(occurrence, t)).toEqual([
      "S-240193",
      "Delivery",
      "Kroger DC standard v3",
    ]);
    expect(detentionSubtitle(occurrence, t)).toBe(
      "The Kroger Co. · appointment Oct 14, 8:00 AM – 9:00 AM",
    );
    expect(placeLine(occurrence)).toBe("Forest Park, GA");
  });

  it("leads a charge waiting on approval with the amount, the rounded time and the rate", () => {
    expect(detentionFigure(pendingCharge(), AFTER, t)).toEqual({
      amount: "$337.50",
      caption: "to bill · 4h 30m at $75.00/hr",
      struck: false,
      quiet: false,
    });
  });

  it("prices a running clock as of now from the server's live price, not the stored zero", () => {
    expect(detentionFigure(accruingCharge(), ACCRUING_NOW, t)).toEqual({
      amount: "$56.25",
      caption: "so far · billing since 8:00 AM",
      struck: false,
      quiet: false,
    });
  });

  it("quiets a running clock still inside its free time", () => {
    expect(detentionFigure(accruingCharge(), at(7), t)).toMatchObject({
      caption: "free time ends 8:00 AM",
      quiet: true,
    });
  });

  it("strikes through a waived charge's amount", () => {
    expect(detentionFigure(waivedCharge(), AFTER, t)).toEqual({
      amount: "$187.50",
      caption: "waived · Oct 14",
      struck: true,
      quiet: false,
    });
  });

  it("says what to check before approving: the approval line and the notice", () => {
    expect(detentionFacts(pendingCharge(), AFTER, t)).toEqual([
      "Over the $250.00 approval line",
      "Notice sent in window",
    ]);
    expect(detentionFacts(accruingCharge(), ACCRUING_NOW, t)).toEqual(["Notice overdue"]);
  });

  it("keeps two facts, the approval line and the notice ahead of a late arrival", () => {
    const detail = toDeskDetention(
      rawDetail({ occurrence: rawOccurrence({ arrivedLate: true, lateByMinutes: 25 }) }),
    );

    expect(detentionFacts(detail, AFTER, t)).toEqual([
      "Over the $250.00 approval line",
      "Notice sent in window",
    ]);
    expect(
      detentionFacts(
        toDeskDetention(
          rawDetail({
            occurrence: rawOccurrence({
              status: "Approved",
              arrivedLate: true,
              lateByMinutes: 25,
            }),
          }),
        ),
        AFTER,
        t,
      ),
    ).toEqual(["Notice sent in window", "Arrived 25m late"]);
  });
});

describe("where it stands", () => {
  it("walks the charge from the dock to the invoice", () => {
    expect(detentionTrack("Pending", t).map((step) => [step.label, step.state])).toEqual([
      ["At the dock", "done"],
      ["Waiting on approval", "now"],
      ["Approved", "next"],
      ["Billed", "next"],
    ]);
  });

  it("strikes the whole way off for a waived, disputed or unbillable charge", () => {
    for (const status of ["Waived", "Disputed", "NotBillable"] as const) {
      expect(detentionTrack(status, t).map((step) => step.state)).toEqual([
        "off",
        "off",
        "off",
        "off",
      ]);
    }
  });

  it("names the pill by where the charge stands", () => {
    expect(detentionStanding("Pending", t)).toEqual({
      label: "Waiting on approval",
      tone: "warn",
      mark: "dot",
    });
    expect(detentionStanding("Billed", t)).toEqual({
      label: "Billed",
      tone: "success",
      mark: "check",
    });
  });

  it("says who approved a charge, or that the policy approved it", () => {
    const approved = (approvedBy: { id: string; name: string } | null) =>
      toDeskDetention(
        rawDetail({
          occurrence: rawOccurrence({ status: "Approved", approvedAt: at(16), approvedBy }),
        }),
      ).occurrence;

    expect(detentionTrackNote(approved({ id: "usr_mia", name: "Mia Chen" }), AFTER, t)).toBe(
      "Approved by Mia Chen Oct 14 · it posts to S-240193 with the next billing run",
    );
    expect(detentionTrackNote(approved(null), AFTER, t)).toBe(
      "Approved automatically · it posts to S-240193 with the next billing run",
    );
  });

  it("counts a running clock from its start to now", () => {
    expect(detentionTrackNote(accruingCharge().occurrence, ACCRUING_NOW, t)).toBe(
      "On the clock 2h 38m · it settles when the truck departs",
    );
  });
});

describe("clockModel", () => {
  it("spans the appointment to the departure, before the clock, free time and billable", () => {
    const model = clockModel(pendingCharge(), AFTER, t);

    expect(model.from).toBe(at(8));
    expect(model.to).toBe(at(15));
    expect(model.live).toBe(false);
    expect(model.spans).toEqual([
      { kind: "pre", from: at(8), to: at(8, 42) },
      { kind: "free", from: at(8, 42), to: at(10, 42) },
      { kind: "bill", from: at(10, 42), to: at(15) },
    ]);
    expect(model.flag).toEqual({ label: "Notice 10:09 AM", at: at(10, 9), tone: "ok" });
    expect(model.dwellMinutes).toBe(378);
    expect(model.freeMinutes).toBe(120);
    expect(model.billable).toEqual({ minutes: 258, rounded: 270 });
  });

  it("drops a moment crowding the one before a row, and names the clock's start only when it is its own moment", () => {
    const { ticks } = clockModel(pendingCharge(), AFTER, t);

    expect(ticks.map((tick) => [tick.label, tick.time, tick.low])).toEqual([
      ["Appointment", "8:00 AM", false],
      ["Arrived", "8:42 AM", true],
      ["Free time ends", "10:42 AM", false],
      ["Departed", "3:00 PM", false],
    ]);
  });

  it("runs a live clock to now, flags the notice due, and counts billable time so far", () => {
    const model = clockModel(accruingCharge(), ACCRUING_NOW, t);

    expect(model.live).toBe(true);
    expect(model.from).toBe(at(5, 51));
    expect(model.to).toBe(ACCRUING_NOW);
    expect(model.ticks.map((tick) => [tick.label, tick.low])).toEqual([
      ["Arrived", false],
      ["Appointment", true],
      ["Free time ends", false],
      ["Now", false],
    ]);
    expect(model.flag).toEqual({ label: "Notice due 7:30 AM", at: at(7, 30), tone: "warn" });
    expect(model.dwellMinutes).toBe(167);
    expect(model.billable).toEqual({ minutes: 38, rounded: 45 });
  });

  it("names an arrival right on the appointment once", () => {
    const detail = toDeskDetention(
      rawDetail({ occurrence: rawOccurrence({ arrivedAt: at(8), clockStartAt: at(8) }) }),
    );

    expect(clockModel(detail, AFTER, t).ticks.map((tick) => tick.label)).toEqual([
      "Arrived",
      "Free time ends",
      "Departed",
    ]);
  });

  it("leaves billable time off a clock that stopped inside its free time", () => {
    const detail = toDeskDetention(
      rawDetail({
        occurrence: rawOccurrence({ departedAt: at(10), clockStopAt: at(10) }),
      }),
    );

    expect(clockModel(detail, AFTER, t).spans.map((span) => span.kind)).toEqual(["pre", "free"]);
  });
});

describe("receiptRows", () => {
  it("lists each step in the engine's words with what it settled on, the charge at its foot", () => {
    const receipt = receiptRows(pendingCharge().occurrence);

    expect(receipt?.rows.map((row) => [row.label, row.value, row.cut])).toEqual([
      ["Policy applied", "", false],
      ["Clock started", "8:42 AM", false],
      ["Clock stopped", "3:00 PM", false],
      ["Total time on site", "6h 18m", false],
      ["Free time deducted", "−2h", true],
      ["Rounded 4h 18m to 4h 30m", "4h 30m", false],
      ["Tier 1", "$337.50", false],
      ["Notice requirement satisfied", "", false],
    ]);
    expect(receipt?.total).toBe("$337.50");
    expect(receipt?.engine).toBe("1.0.0");
  });

  it("is nothing for an occurrence with no trace", () => {
    expect(receiptRows(accruingCharge().occurrence)).toBeNull();
  });
});

describe("the money", () => {
  it("splits the charge between the worker's pay and what is kept", () => {
    expect(paySplit(pendingCharge().occurrence, t)).toEqual({
      bill: 337.5,
      pay: 130,
      margin: 207.5,
      payNote: "3h 15m payable",
      payShare: 130 / 337.5,
    });
  });
});

describe("the notice", () => {
  it("says what the policy asks and what happens without it", () => {
    expect(noticeRule(pendingCharge().occurrence, t)).toBe(
      "Kroger DC standard asks for notice 30m before free time runs out; without it the charge is held for review.",
    );
    expect(noticeRule(accruingCharge().occurrence, t)).toBe(
      "Cold storage pickups asks for notice 30m before free time runs out; without it the charge won't bill.",
    );
  });

  it("says nothing for a policy that asks for no notice", () => {
    const detail = toDeskDetention(
      rawDetail({
        occurrence: rawOccurrence({
          policySnapshot: snapshotJson({ notificationRequirement: "None" }),
        }),
      }),
    );

    expect(noticeRule(detail.occurrence, t)).toBe("");
  });

  it("warns of an overdue notice, by how much and by when to send it", () => {
    expect(noticeSlip(accruingCharge().occurrence, ACCRUING_NOW, t)).toEqual({
      title: "Notice overdue by 1h 8m",
      body: "It was due at 7:30 AM. Send it before 8:45 AM, or this policy holds the charge back and it won't bill.",
    });
    expect(noticeSlip(pendingCharge().occurrence, AFTER, t)).toBeNull();
  });

  it("reads a pending notice as due until its time passes, then overdue", () => {
    const { occurrence } = accruingCharge();

    expect(noticeReading(occurrence, at(7))).toBe("due");
    expect(noticeReading(occurrence, at(7, 31))).toBe("overdue");
  });

  it("lists each notice with where it went and whether it landed in its window", () => {
    expect(noticeRows(pendingCharge().notices, t)).toEqual([
      {
        key: "dtn_warning",
        title: "Warning notice · Delivered",
        to: "To dock-atl@kroger.com and ap@kroger.com",
        when: "Oct 14, 10:09 AM",
        tone: "ok",
      },
    ]);
  });

  it("gives a bounced notice's reason in place of its recipients", () => {
    const rows = noticeRows(
      [
        {
          ...pendingCharge().notices[0],
          deliveryStatus: "Bounced",
          failureReason: "Mailbox ap@kroger.com does not exist",
        },
      ],
      t,
    );

    expect(rows[0]).toMatchObject({ to: "Mailbox ap@kroger.com does not exist", tone: "warn" });
  });
});

describe("proofReading", () => {
  it("scores the claim, says what each short factor needs, and orders the evidence by time", () => {
    const proof = proofReading(pendingCharge(), t);

    expect(proof.score).toBe(86);
    expect(proof.band).toBe("of 100 · strong");
    expect(proof.weak).toBe(false);
    expect(proof.factors.map((factor) => [factor.label, factor.points, factor.remedy])).toEqual([
      ["Arrival timestamp", "20/20", ""],
      ["Customer notice", "25/25", ""],
      ["Supporting document", "11/25", "Attach the POD with the receiver's in and out times."],
    ]);
    expect(proof.events.map((event) => [event.title, event.when])).toEqual([
      ["Arrival · Telematics", "8:42 AM"],
      ["Notice · System", "10:09 AM"],
      ["Departure · Telematics", "3:00 PM"],
    ]);
    expect(proof.aside).toBe("3 records · chain verified");
  });

  it("says when the evidence chain no longer verifies", () => {
    const proof = proofReading(
      toDeskDetention(
        rawDetail({
          collectability: { ...rawDetail().collectability, chainValid: false },
        }),
      ),
      t,
    );

    expect(proof.aside).toBe("3 records · chain broken");
    expect(proof.chainValid).toBe(false);
  });
});

describe("detentionActions", () => {
  it("leads a charge waiting on approval with approving it", () => {
    expect(detentionActions(pendingCharge().occurrence)).toEqual({
      primary: "approve",
      more: ["dispute", "waive"],
    });
  });

  it("leads a running clock with the overdue notice and offers no dispute of nothing billed", () => {
    expect(detentionActions(accruingCharge().occurrence)).toEqual({
      primary: "notice",
      more: ["waive"],
    });
  });

  it("offers no waiver of a charge already waived", () => {
    expect(detentionActions(waivedCharge().occurrence)).toEqual({
      primary: null,
      more: ["dispute"],
    });
  });

  it("keeps the notice beside approval when both are owed", () => {
    const detail = toDeskDetention(
      rawDetail({ occurrence: rawOccurrence({ notificationStatus: "Missed" }) }),
    );

    expect(detentionActions(detail.occurrence)).toEqual({
      primary: "approve",
      more: ["notice", "dispute", "waive"],
    });
  });
});

describe("settledSlip", () => {
  it("keeps the waiver as written with its reason, who, when and how far it had got", () => {
    expect(settledSlip(waivedCharge().occurrence, t)).toEqual({
      title: "Why it was waived",
      note: "Waived for the Q4 volume commitment.",
      facts: [
        { key: "reason", label: "Reason", value: "Customer goodwill" },
        { key: "by", label: "Waived by", value: "Dana Ruiz" },
        { key: "when", label: "When", value: "Oct 14, 5:12 PM" },
        { key: "reached", label: "Had reached", value: "Waiting on approval" },
      ],
    });
  });

  it("is nothing for a charge still on its way", () => {
    expect(settledSlip(pendingCharge().occurrence, t)).toBeNull();
  });
});
