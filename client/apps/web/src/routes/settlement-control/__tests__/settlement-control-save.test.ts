import { afterEach, describe, expect, it, vi } from "vitest";
import { updateSettlementControl } from "@/lib/graphql/driver-settlement";
import { savedSettlementControl } from "./settlement-control-fixture";

const requestGraphQL = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL,
}));

afterEach(() => requestGraphQL.mockReset());

describe("updateSettlementControl", () => {
  // UpdateSettlementControlInput has no id or tenant fields; GraphQL refuses an input that
  // carries one, so the record's own keys never reach it.
  it("sends every setting and the version, and nothing that identifies the record", async () => {
    requestGraphQL.mockResolvedValue({ updateSettlementControl: { id: "stlc_01", version: 3 } });

    await updateSettlementControl(savedSettlementControl);

    expect(requestGraphQL).toHaveBeenCalledTimes(1);
    expect(requestGraphQL.mock.calls[0][0]).toMatchObject({
      operationName: "UpdateSettlementControl",
      variables: {
        input: {
          version: 2,
          payPeriodFrequency: "Weekly",
          periodEndDayOfWeek: 6,
          payDelayDays: 5,
          payTrigger: "ShipmentDelivered",
          autoGenerateBatches: false,
          autoApproveClean: false,
          autoAttachAccruals: false,
          requireSeparateApprover: true,
          autoPostOnApprove: false,
          allowNegativeNet: false,
          varianceThresholdPct: "25",
          varianceLookbackWeeks: 8,
          defaultEscrowInterestRate: "0",
          escrowInterestFrequencyMonths: 3,
        },
      },
    });
    expect(Object.keys(requestGraphQL.mock.calls[0][0].variables.input)).toHaveLength(15);
  });

  // The mutation answers with the id and the new version only; the form resets to what this
  // returns, so the next save has to carry that version.
  it("returns the record as saved, under the version the server gave it", async () => {
    requestGraphQL.mockResolvedValue({ updateSettlementControl: { id: "stlc_01", version: 3 } });

    expect(await updateSettlementControl(savedSettlementControl)).toEqual({
      ...savedSettlementControl,
      version: 3,
    });
  });
});
