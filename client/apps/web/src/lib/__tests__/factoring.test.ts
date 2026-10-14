import { describe, expect, it } from "vitest";
import {
  factoringItemActions,
  factoringNoaDisplayStatus,
  factoringNoaPhase,
  factoringSubmissionAwaitsFunding,
  factoringSubmissionPhase,
  factoringVerdictPhase,
  formatBasisPoints,
  formatMinutesOfDay,
  groupInvoicesByFactor,
  parseMinutesOfDay,
  percentToBasisPoints,
  readyOverADay,
} from "@/lib/factoring";

const DAY = 24 * 60 * 60;

describe("factoring rates", () => {
  it("reads basis points as the percentage the contract states", () => {
    expect(formatBasisPoints(8500)).toBe("85%");
    expect(formatBasisPoints(275)).toBe("2.75%");
    expect(formatBasisPoints(0)).toBe("0%");
  });

  it("turns a typed percentage into whole basis points without float drift", () => {
    expect(percentToBasisPoints(2.75)).toBe(275);
    expect(percentToBasisPoints(0.29)).toBe(29);
    expect(percentToBasisPoints(100)).toBe(10_000);
  });
});

describe("factoring submission time", () => {
  it("round-trips minutes after midnight through a 24-hour clock time", () => {
    expect(formatMinutesOfDay(0)).toBe("00:00");
    expect(formatMinutesOfDay(570)).toBe("09:30");
    expect(formatMinutesOfDay(1439)).toBe("23:59");
    expect(parseMinutesOfDay("09:30")).toBe(570);
    expect(parseMinutesOfDay("23:59")).toBe(1439);
  });

  it("refuses a value that is not a time of day", () => {
    expect(parseMinutesOfDay("24:00")).toBeNull();
    expect(parseMinutesOfDay("9:75")).toBeNull();
    expect(parseMinutesOfDay("")).toBeNull();
  });
});

describe("notice of assignment status", () => {
  const now = 1_800_000_000;

  it("reads an acknowledged notice whose signed copy has lapsed as expired", () => {
    expect(factoringNoaDisplayStatus({ status: "Acknowledged", expiresAt: now }, now)).toBe(
      "Expired",
    );
    expect(factoringNoaPhase("Expired")).toBe("failed");
  });

  it("keeps an acknowledged notice in force until it expires, or forever without an expiry", () => {
    expect(factoringNoaDisplayStatus({ status: "Acknowledged", expiresAt: now + 1 }, now)).toBe(
      "Acknowledged",
    );
    expect(factoringNoaDisplayStatus({ status: "Acknowledged", expiresAt: null }, now)).toBe(
      "Acknowledged",
    );
  });

  it("never reads an expiry onto a notice that was not acknowledged", () => {
    expect(factoringNoaDisplayStatus({ status: "Sent", expiresAt: now - 1 }, now)).toBe("Sent");
    expect(factoringNoaDisplayStatus({ status: "Released", expiresAt: now - 1 }, now)).toBe(
      "Released",
    );
  });

  it("puts a sent notice in front of a person and a released one out of the way", () => {
    expect(factoringNoaPhase("Sent")).toBe("awaiting");
    expect(factoringNoaPhase("Acknowledged")).toBe("complete");
    expect(factoringNoaPhase("Released")).toBe("closed");
  });
});

describe("factoring readiness", () => {
  it("colours each verdict by what the biller does next", () => {
    expect(factoringVerdictPhase("Ready")).toBe("complete");
    expect(factoringVerdictPhase("NotReady")).toBe("attention");
    expect(factoringVerdictPhase("Excluded")).toBe("closed");
  });

  it("counts an invoice as ready over a day from the full day on, and never when not ready", () => {
    const now = 1_800_000_000;
    expect(readyOverADay(now - DAY, now)).toBe(true);
    expect(readyOverADay(now - DAY + 1, now)).toBe(false);
    expect(readyOverADay(null, now)).toBe(false);
  });
});

describe("groupInvoicesByFactor", () => {
  const row = (invoiceId: string, factorId: string | null, name?: string) => ({
    invoiceId,
    factoringCompanyId: factorId,
    factoringCompany: factorId ? { id: factorId, name: name ?? factorId } : null,
  });

  it("puts the factor most of the selection is with first, keeping each factor's invoices in selection order", () => {
    const groups = groupInvoicesByFactor([
      row("inv_1", "fac_a", "Apex Capital"),
      row("inv_2", "fac_b", "RTS Financial"),
      row("inv_3", "fac_b", "RTS Financial"),
      row("inv_4", "fac_a", "Apex Capital"),
      row("inv_5", "fac_b", "RTS Financial"),
    ]);

    expect(groups).toEqual([
      {
        factoringCompanyId: "fac_b",
        name: "RTS Financial",
        invoiceIds: ["inv_2", "inv_3", "inv_5"],
      },
      { factoringCompanyId: "fac_a", name: "Apex Capital", invoiceIds: ["inv_1", "inv_4"] },
    ]);
  });

  it("orders factors with as many invoices by name, and leaves out invoices with no factor", () => {
    const groups = groupInvoicesByFactor([
      row("inv_1", "fac_z", "Zeal Funding"),
      row("inv_2", null),
      row("inv_3", "fac_a", "Apex Capital"),
    ]);

    expect(groups.map((group) => group.factoringCompanyId)).toEqual(["fac_a", "fac_z"]);
    expect(groups.flatMap((group) => group.invoiceIds)).toEqual(["inv_3", "inv_1"]);
  });

  it("names a factor by its id when the row did not carry the factor", () => {
    expect(groupInvoicesByFactor([{ invoiceId: "inv_1", factoringCompanyId: "fac_a" }])).toEqual([
      { factoringCompanyId: "fac_a", name: "fac_a", invoiceIds: ["inv_1"] },
    ]);
  });
});

describe("factoring submission phases", () => {
  it("reads a failed schedule as failed and a cancelled one as closed", () => {
    expect(factoringSubmissionPhase("Sending")).toBe("queued");
    expect(factoringSubmissionPhase("Failed")).toBe("failed");
    expect(factoringSubmissionPhase("Cancelled")).toBe("closed");
  });

  it("reads a delivered schedule as awaiting the factor until every invoice is decided", () => {
    expect(factoringSubmissionPhase("Sent")).toBe("awaiting");
    expect(factoringSubmissionPhase("PartiallyFunded")).toBe("awaiting");
    expect(factoringSubmissionPhase("Funded")).toBe("complete");
    expect(factoringSubmissionPhase("Rejected")).toBe("failed");
  });

  it("reads a submitted invoice as active, between ready and closed", () => {
    expect(factoringVerdictPhase("Submitted")).toBe("active");
  });
});

describe("factoring item actions", () => {
  it("offers a funded invoice's release, and its chargeback only under recourse", () => {
    expect(factoringItemActions("Funded", true)).toEqual(["release", "chargeback"]);
    expect(factoringItemActions("Funded", false)).toEqual(["release"]);
  });

  it("offers a charged-back invoice only the reversal of its chargeback", () => {
    expect(factoringItemActions("ChargedBack", true)).toEqual(["reverse"]);
    expect(factoringItemActions("ChargedBack", false)).toEqual(["reverse"]);
  });

  it("offers nothing on an invoice the factor has not funded or is done with", () => {
    for (const status of ["Pending", "Released", "Rejected", "Settled"] as const) {
      expect(factoringItemActions(status, true)).toEqual([]);
    }
  });

  it("lets funding be recorded only on a delivered schedule the factor has not finished", () => {
    expect(
      (
        ["Sending", "Sent", "PartiallyFunded", "Funded", "Rejected", "Failed", "Cancelled"] as const
      ).filter(factoringSubmissionAwaitsFunding),
    ).toEqual(["Sent", "PartiallyFunded"]);
  });
});
