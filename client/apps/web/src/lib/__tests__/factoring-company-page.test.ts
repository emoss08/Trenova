import { describe, expect, it } from "vitest";
import { factorRemitLines, standardNoticeOfAssignment } from "../factoring";
import { factoringWorkspacePath } from "@/routes/factoring/use-factoring-state";
import { factoringCompanyPath } from "@/routes/factoring-company/use-factoring-company-state";

describe("standardNoticeOfAssignment", () => {
  it("prints the server's standard notice word for word", () => {
    expect(standardNoticeOfAssignment("  RTS Financial ")).toBe(
      "NOTICE OF ASSIGNMENT: This invoice has been assigned to, and is payable only to, RTS Financial. Payment to any other party will not discharge your obligation. Remit payment to the address below and direct any dispute to RTS Financial.",
    );
  });
});

describe("factoringWorkspacePath", () => {
  it("narrows the workspace to one factor when a factor's page opens it", () => {
    expect(factoringWorkspacePath("funded", "fac_1")).toBe(
      "/billing/factoring?view=funded&factor=fac_1",
    );
    expect(factoringWorkspacePath("ready")).toBe("/billing/factoring?view=ready");
  });
});

describe("factoringCompanyPath", () => {
  it("opens a factor's page, on a tab with a record picked out when a link names one", () => {
    expect(factoringCompanyPath("fac_1")).toBe(
      "/billing/configuration-files/factoring-companies/fac_1",
    );
    expect(factoringCompanyPath("fac_1", { tab: "agreement", agreement: "facagr_9" })).toBe(
      "/billing/configuration-files/factoring-companies/fac_1?tab=agreement&agreement=facagr_9",
    );
    expect(factoringCompanyPath("fac_1", { tab: "customers", noa: "facnoa_2" })).toBe(
      "/billing/configuration-files/factoring-companies/fac_1?tab=customers&noa=facnoa_2",
    );
  });
});

describe("factorRemitLines", () => {
  it("prints the address, each instruction line, then email and phone, as the server does", () => {
    expect(
      factorRemitLines({
        line1: "PO Box 29237",
        city: "Jacksonville",
        state: "FL",
        postalCode: "32202",
        country: "United States",
        remittanceInstructions: "ACH routing 567272913\n\nAccount ending 0395",
        email: " funding@factor.example ",
        phoneNumber: "+17338043532",
      }),
    ).toEqual([
      "PO Box 29237",
      "Jacksonville, FL 32202",
      "United States",
      "ACH routing 567272913",
      "Account ending 0395",
      "Email: funding@factor.example",
      "Phone: +17338043532",
    ]);
  });

  it("prints nothing for a factor with no remit details", () => {
    expect(factorRemitLines({ email: "  ", remittanceInstructions: "" })).toEqual([]);
  });
});
