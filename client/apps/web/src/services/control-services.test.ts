import { describe, expect, it, vi } from "vitest";

import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { AccountingControl } from "@/types/accounting-control";
import type { DispatchControl } from "@/types/dispatch-control";

import { toAccountingControlInput } from "./accounting-control";
import { DataEntryControlService } from "./data-entry-control";
import { toDispatchControlInput } from "./dispatch-control";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

describe("control services", () => {
  it("sends money as decimal strings and unset accounts as null", () => {
    const input = toAccountingControlInput({
      version: 4,
      reconciliationToleranceAmount: 12.5,
      defaultRevenueAccountId: "gla_01",
      defaultCashAccountId: null,
      defaultArAccountId: "",
    } as unknown as AccountingControl);

    expect(input.version).toBe(4);
    expect(input.reconciliationToleranceAmount).toBe("12.5");
    expect(input.defaultRevenueAccountId).toBe("gla_01");
    expect(input.defaultCashAccountId).toBeNull();
    expect(input.defaultArAccountId).toBeNull();
  });

  it("keeps a cleared margin floor null and sends scoring weights whole", () => {
    const input = toDispatchControlInput({
      version: 1,
      autoAssignConfidenceThreshold: 0.8,
      tenderEscalationRateIncreasePercent: 5,
      tenderCounterMarginFloorPercent: null,
      scoringWeights: { deadhead: 2 },
    } as unknown as DispatchControl);

    expect(input.autoAssignConfidenceThreshold).toBe("0.8");
    expect(input.tenderCounterMarginFloorPercent).toBeNull();
    expect(input.scoringWeights).toEqual({ deadhead: 2 });
  });

  it("reads a control through its GraphQL query and forwards the signal", async () => {
    const control = {
      id: "dec_1",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      codeCase: "Upper",
      nameCase: "TitleCase",
      emailCase: "Lower",
      cityCase: "AsEntered",
      version: 2,
      createdAt: 1_780_000_000,
      updatedAt: 1_780_000_000,
    };
    vi.mocked(requestGraphQL).mockResolvedValueOnce({ dataEntryControl: control });
    const controller = new AbortController();

    const result = await new DataEntryControlService().get({ signal: controller.signal });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "DataEntryControl", signal: controller.signal }),
    );
    expect(result.codeCase).toBe("Upper");
  });
});
