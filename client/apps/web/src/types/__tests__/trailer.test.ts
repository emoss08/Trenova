import { describe, expect, it } from "vitest";
import { trailerSchema } from "@/types/trailer";

// GraphQL types customFields as a nullable JSON object, so a trailer with no
// custom field values comes back as null rather than an empty object. The form
// resolver parses the row it was handed, so a schema that only allows undefined
// blocks every save on such a trailer.

const base = {
  id: "trl_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  organizationId: "org_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  businessUnitId: "bu_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  version: 1,
  status: "Available" as const,
  code: "TRL-118",
  equipmentTypeId: "et_1",
  equipmentManufacturerId: "em_1",
  model: "Reefer 53",
  make: "Utility",
  year: 2021,
};

describe("trailerSchema customFields", () => {
  it("accepts the null a trailer without custom field values arrives with", () => {
    const parsed = trailerSchema.safeParse({ ...base, customFields: null });

    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    expect(parsed.data?.customFields).toBeNull();
  });

  it("accepts an omitted key and a populated record", () => {
    expect(trailerSchema.safeParse(base).success).toBe(true);

    const parsed = trailerSchema.safeParse({
      ...base,
      customFields: { doorType: "Roll-up", palletPositions: 26 },
    });
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    expect(parsed.data?.customFields).toEqual({ doorType: "Roll-up", palletPositions: 26 });
  });

  it("still rejects a value that is not a record", () => {
    expect(trailerSchema.safeParse({ ...base, customFields: "doorType=Roll-up" }).success).toBe(
      false,
    );
  });
});
