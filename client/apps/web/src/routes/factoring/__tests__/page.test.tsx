import { factoring } from "@/lib/queries/factoring";
import { describe, expect, it } from "vitest";
import { prefetch } from "../page";

describe("prefetch", () => {
  it("warms the factoring summary every tab of the page opens with", () => {
    const args = {
      params: {},
      request: new Request("http://localhost/billing/factoring?view=schedules"),
      context: undefined,
    } as unknown as Parameters<typeof prefetch>[0];

    expect(prefetch(args)).toEqual([
      expect.objectContaining({ queryKey: factoring.summary().queryKey }),
    ]);
  });
});
