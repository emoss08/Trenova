import { beforeEach, describe, expect, it, vi } from "vitest";
import { confirmAccountingMappings, MAX_CONFIRM_BATCH } from "../accounting-sync";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));

describe("confirmAccountingMappings", () => {
  beforeEach(() => {
    mocks.requestGraphQL.mockReset();
    mocks.requestGraphQL.mockResolvedValue({ confirmAccountingMappings: [] });
  });

  it("sends a long selection in batches the server accepts", async () => {
    const items = Array.from({ length: MAX_CONFIRM_BATCH * 2 + 50 }, (_, index) => ({
      id: `acctm_${index}`,
      externalId: String(index),
    }));

    await confirmAccountingMappings(items);

    const sizes = mocks.requestGraphQL.mock.calls.map(
      ([request]) => (request as { variables: { input: unknown[] } }).variables.input.length,
    );
    expect(sizes).toEqual([MAX_CONFIRM_BATCH, MAX_CONFIRM_BATCH, 50]);
  });

  it("sends nothing when nothing is ticked", async () => {
    await confirmAccountingMappings([]);

    expect(mocks.requestGraphQL).not.toHaveBeenCalled();
  });
});
