import type { FactoringNoa } from "@/lib/graphql/factoring-noa";
import { describe, expect, it } from "vitest";
import { currentFactoringNoa } from "../customer-factoring-noa";

function noa(id: string, status: FactoringNoa["status"]): FactoringNoa {
  return {
    id,
    customerId: "cus_1",
    factoringCompanyId: "fac_1",
    status,
    sentAt: null,
    sentTo: null,
    acknowledgedAt: null,
    signedDocumentId: null,
    expiresAt: null,
    releasedAt: null,
    releaseReason: null,
    notes: null,
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    customer: null,
    factoringCompany: null,
  };
}

describe("the notice that governs a factored customer", () => {
  it("is the newest notice not yet released, skipping released ones listed before it", () => {
    expect(
      currentFactoringNoa([noa("n3", "Released"), noa("n2", "Sent"), noa("n1", "Released")])?.id,
    ).toBe("n2");
  });

  it("is none once every notice has been released, so a new one is drafted", () => {
    expect(currentFactoringNoa([noa("n2", "Released"), noa("n1", "Released")])).toBeNull();
    expect(currentFactoringNoa([])).toBeNull();
  });
});
