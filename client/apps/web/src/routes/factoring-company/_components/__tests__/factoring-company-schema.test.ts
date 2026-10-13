import { factoringCompanySchema } from "@/types/factoring-company";
import { describe, expect, it } from "vitest";

const valid = {
  status: "Active",
  code: "RTS",
  name: "RTS Financial",
  addressLine1: "PO Box 840267",
  city: "Dallas",
  stateId: "st_tx",
  postalCode: "75284",
};

describe("factoringCompanySchema", () => {
  it("accepts a factor with only its required details", () => {
    expect(factoringCompanySchema.safeParse(valid).success).toBe(true);
  });

  it("treats a factor as recourse unless it is marked non-recourse", () => {
    expect(factoringCompanySchema.parse(valid).nonRecourse).toBe(false);
    expect(factoringCompanySchema.parse({ ...valid, nonRecourse: true }).nonRecourse).toBe(true);
  });

  it("accepts the nulls a table row carries for optional text", () => {
    const parsed = factoringCompanySchema.safeParse({
      ...valid,
      addressLine2: null,
      email: null,
      remittanceInstructions: null,
      noticeOfAssignment: null,
    });
    expect(parsed.success).toBe(true);
  });

  it("refuses a bad email, a long code and a missing remit-to address", () => {
    const parsed = factoringCompanySchema.safeParse({
      ...valid,
      email: "not-an-email",
      code: "TOO-LONG-CODE",
      addressLine1: "",
    });
    expect(parsed.success).toBe(false);
    const fields = parsed.error?.issues.map((issue) => issue.path.join("."));
    expect(fields).toEqual(expect.arrayContaining(["email", "code", "addressLine1"]));
  });

  it("refuses a notice of assignment over 2,000 characters", () => {
    const parsed = factoringCompanySchema.safeParse({
      ...valid,
      noticeOfAssignment: "x".repeat(2001),
    });
    expect(parsed.success).toBe(false);
  });
});
