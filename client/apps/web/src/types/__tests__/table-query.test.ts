import { composedTableQuerySchema } from "@/types/table-query";
import { describe, expect, it } from "vitest";

/**
 * A composed filter is applied as it arrives, so an operator the table cannot
 * apply must fail the parse rather than land as a chip that filters nothing —
 * the question would come back looking answered with its one condition gone.
 */
describe("composedTableQuerySchema", () => {
  it("accepts the operators and directions the table applies", () => {
    const parsed = composedTableQuerySchema.parse({
      fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
      sort: [{ field: "createdAt", direction: "desc" }],
    });

    expect(parsed.fieldFilters[0]?.operator).toBe("eq");
    expect(parsed.sort[0]?.direction).toBe("desc");
  });

  it("refuses an operator the table cannot apply", () => {
    expect(
      composedTableQuerySchema.safeParse({
        fieldFilters: [{ field: "status", operator: "sounds_like", value: "x" }],
      }).success,
    ).toBe(false);
  });

  it("refuses a direction the table cannot sort by", () => {
    expect(
      composedTableQuerySchema.safeParse({ sort: [{ field: "createdAt", direction: "up" }] })
        .success,
    ).toBe(false);
  });
});
