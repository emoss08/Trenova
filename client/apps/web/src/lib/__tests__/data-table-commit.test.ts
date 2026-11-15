import { describe, expect, it, vi } from "vitest";
import { commitRowField } from "../data-table-commit";

describe("commitRowField", () => {
  it("saves one field of a row through the table's own edit commit", async () => {
    const onCellEditCommit = vi.fn().mockResolvedValue(undefined);
    const original = { id: "st_1", code: "FTL", color: "#ff75c3" };

    await commitRowField(
      { options: { onCellEditCommit } },
      { id: "st_1", original },
      "color",
      "#2e8b57",
    );

    expect(onCellEditCommit).toHaveBeenCalledExactlyOnceWith({
      rowId: "st_1",
      columnId: "color",
      value: "#2e8b57",
      previousValue: "#ff75c3",
      row: original,
    });
  });

  it("passes on a refusal from the table's commit", async () => {
    const onCellEditCommit = vi.fn().mockRejectedValue(new Error("Code is required."));

    await expect(
      commitRowField(
        { options: { onCellEditCommit } },
        { id: "st_1", original: { code: "" } },
        "code",
        "",
      ),
    ).rejects.toThrow("Code is required.");
  });
});
