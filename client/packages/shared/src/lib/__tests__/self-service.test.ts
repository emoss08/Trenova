import { describe, expect, it } from "vitest";
import {
  changeRequestTone,
  compliancePercent,
  describeChanges,
  orderPoliciesForSigning,
  policyAudienceLabel,
  policyStanding,
  signatureMatches,
} from "../self-service";

describe("signatureMatches", () => {
  // A signature is a claim that a specific person agreed, and "ok" is not a
  // person. The server enforces this too; the client says so before the tap.
  it("accepts the worker's own name however it is cased or spaced", () => {
    expect(signatureMatches("Ada Byron", "Ada", "Byron")).toBe(true);
    expect(signatureMatches("  ada   BYRON ", "Ada", "Byron")).toBe(true);
  });

  it("refuses anything else", () => {
    expect(signatureMatches("ok", "Ada", "Byron")).toBe(false);
    expect(signatureMatches("Ada", "Ada", "Byron")).toBe(false);
    expect(signatureMatches("", "Ada", "Byron")).toBe(false);
  });
});

describe("policyStanding", () => {
  it("is signed once there is an acknowledgement on the version in force", () => {
    expect(policyStanding({ acknowledgedAt: 1_800_000_000, requiresSignature: true })).toBe(
      "signed",
    );
  });

  it("is outstanding until then", () => {
    expect(policyStanding({ acknowledgedAt: null, requiresSignature: true })).toBe("outstanding");
  });

  // A policy that only needs reading is a lighter ask, and the card should
  // look lighter for it.
  it("distinguishes a read receipt from a signature", () => {
    expect(policyStanding({ acknowledgedAt: null, requiresSignature: false })).toBe("unread");
    expect(policyStanding({ acknowledgedAt: 1, requiresSignature: false })).toBe("read");
  });
});

describe("describeChanges", () => {
  it("reads as a sentence a manager can scan", () => {
    expect(
      describeChanges([
        { field: "city", label: "City", from: "Springfield", to: "Shelbyville" },
        { field: "phoneNumber", label: "Phone", from: "", to: "555-0100" },
      ]),
    ).toEqual(["City: Springfield → Shelbyville", "Phone: (blank) → 555-0100"]);
  });

  it("says when a field is being cleared", () => {
    expect(
      describeChanges([{ field: "addressLine2", label: "Address line 2", from: "Apt 2", to: "" }]),
    ).toEqual(["Address line 2: Apt 2 → (blank)"]);
  });
});

describe("policyAudienceLabel", () => {
  it("names every audience", () => {
    expect(policyAudienceLabel("All")).toBe("Everyone");
    expect(policyAudienceLabel("Employees")).toBe("Employees");
    expect(policyAudienceLabel("Contractors")).toBe("Owner-operators");
  });

  it("falls back to the raw value rather than rendering nothing", () => {
    expect(policyAudienceLabel("Something")).toBe("Something");
  });
});

describe("changeRequestTone", () => {
  it("gives every status a tone", () => {
    for (const status of ["Pending", "Approved", "Rejected", "Withdrawn"]) {
      expect(changeRequestTone(status).label).toBeTruthy();
    }
  });

  it("falls back rather than rendering nothing", () => {
    expect(changeRequestTone("Other")).toEqual(changeRequestTone("Pending"));
  });
});

describe("orderPoliciesForSigning", () => {
  const policy = (id: string, requiresSignature: boolean, acknowledgedAt: number | null) => ({
    id,
    requiresSignature,
    acknowledgedAt,
  });

  // The driver opens the card to find out what is asked of them. What they
  // have already done belongs underneath, however the server happened to
  // order the rows.
  it("puts what still needs doing before what is done, signatures first", () => {
    const ordered = orderPoliciesForSigning([
      policy("signed", true, 1_800_000_000),
      policy("read", false, 1_800_000_000),
      policy("unread", false, null),
      policy("outstanding", true, null),
    ]);
    expect(ordered.map((row) => row.id)).toEqual(["outstanding", "unread", "signed", "read"]);
  });

  it("keeps the server's order inside a group", () => {
    const ordered = orderPoliciesForSigning([
      policy("a", true, null),
      policy("b", true, null),
      policy("c", false, null),
    ]);
    expect(ordered.map((row) => row.id)).toEqual(["a", "b", "c"]);
  });
});

describe("compliancePercent", () => {
  it("is the signed share of everybody the policy applies to", () => {
    expect(compliancePercent(3, 1)).toBe(75);
    expect(compliancePercent(1, 2)).toBe(33);
  });

  // A policy that applies to nobody yet is not "0% compliant"; it is simply
  // not owed by anyone.
  it("is zero when nobody is in scope rather than dividing by nothing", () => {
    expect(compliancePercent(0, 0)).toBe(0);
  });
});
