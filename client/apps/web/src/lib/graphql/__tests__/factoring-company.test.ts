import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FactoringCompany } from "@/types/factoring-company";
import { createFactoringCompany, updateFactoringCompany } from "../factoring-company";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

const factor = {
  id: "fac_1",
  version: 4,
  status: "Active",
  code: "RTS",
  name: "RTS Financial",
  addressLine1: "9300 Metcalf Ave",
  addressLine2: "",
  city: "Overland Park",
  stateId: "us_ks",
  postalCode: "66212",
  email: "",
  phoneNumber: "913-555-0100",
  remittanceInstructions: "Remit to lockbox 1234",
  noticeOfAssignment: "",
  notes: null,
  nonRecourse: true,
} as FactoringCompany;

const expectedInput = {
  status: "Active",
  code: "RTS",
  name: "RTS Financial",
  addressLine1: "9300 Metcalf Ave",
  addressLine2: null,
  city: "Overland Park",
  stateId: "us_ks",
  postalCode: "66212",
  email: null,
  phoneNumber: "913-555-0100",
  remittanceInstructions: "Remit to lockbox 1234",
  noticeOfAssignment: null,
  nonRecourse: true,
  notes: null,
};

describe("factoring company mutations", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("creates a factor from the form, sending blank optional text as null", async () => {
    requestGraphQL.mockResolvedValue({ createFactoringCompany: { id: "fac_2", version: 1 } });

    const created = await createFactoringCompany(factor);

    expect(requestGraphQL).toHaveBeenCalledWith({
      document: expect.anything(),
      operationName: "CreateFactoringCompany",
      variables: { input: expectedInput },
    });
    expect(created).toEqual({ id: "fac_2", version: 1 });
  });

  it("updates a factor by id at the version the form was loaded with", async () => {
    requestGraphQL.mockResolvedValue({ updateFactoringCompany: { id: "fac_1", version: 5 } });

    await updateFactoringCompany("fac_1", factor);

    expect(requestGraphQL).toHaveBeenCalledWith({
      document: expect.anything(),
      operationName: "UpdateFactoringCompany",
      variables: { input: { id: "fac_1", version: 4, factoringCompany: expectedInput } },
    });
  });
});
