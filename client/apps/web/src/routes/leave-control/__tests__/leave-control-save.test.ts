import { updateLeaveControl } from "@/lib/graphql/worker-leave";
import { afterEach, describe, expect, it, vi } from "vitest";
import { savedLeaveControl } from "./leave-control-fixture";

const requestGraphQL = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL,
}));

afterEach(() => requestGraphQL.mockReset());

describe("updateLeaveControl", () => {
  // UpdateLeaveControlInput holds the version the form read and the seven settings; the
  // server refuses a save over a newer version, and GraphQL refuses an input carrying the id.
  it("sends the version it read and every setting, and not the record's id", async () => {
    requestGraphQL.mockResolvedValue({
      updateLeaveControl: { id: "lctl_01", measurementMethod: "RollingBackward", version: 5 },
    });

    await updateLeaveControl({ ...savedLeaveControl, workweekHours: "37.5" });

    expect(requestGraphQL).toHaveBeenCalledTimes(1);
    expect(requestGraphQL.mock.calls[0][0]).toMatchObject({
      operationName: "UpdateLeaveControl",
      variables: {
        input: {
          version: 4,
          measurementMethod: "RollingBackward",
          entitlementWeeks: "12",
          militaryCaregiverWeeks: "26",
          workweekHours: "37.5",
          eligibilityMonths: 12,
          eligibilityHours: 1250,
          certificationDueDays: 15,
        },
      },
    });
    expect(Object.keys(requestGraphQL.mock.calls[0][0].variables.input)).toHaveLength(8);
    expect(requestGraphQL.mock.calls[0][0].variables.input).not.toHaveProperty("id");
  });

  // The mutation answers with the id, the method and the new version only; the form resets
  // to what this returns.
  it("returns the record as saved, under the version the server gave it", async () => {
    requestGraphQL.mockResolvedValue({
      updateLeaveControl: { id: "lctl_01", measurementMethod: "RollingBackward", version: 5 },
    });

    expect(await updateLeaveControl(savedLeaveControl)).toEqual({
      ...savedLeaveControl,
      version: 5,
    });
  });
});
