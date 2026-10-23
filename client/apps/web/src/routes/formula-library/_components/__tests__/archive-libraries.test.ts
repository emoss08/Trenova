import { ApiRequestError } from "@trenova/shared/lib/api";
import { describe, expect, it, vi } from "vitest";
import { archiveLibraries } from "../archive-libraries";

const inUse = "Templates still import org.fees: Per Mile v3. Confirm to archive it anyway";

function needsConfirmation() {
  return new ApiRequestError(422, {
    type: "validation-error",
    title: "Validation failed",
    status: 422,
    errors: [{ field: "confirmInUse", message: inUse, code: "INVALID" }],
  });
}

function forbidden() {
  return new ApiRequestError(403, {
    type: "authorization-error",
    title: "Forbidden",
    status: 403,
    detail: "Archiving a library templates import needs approve permission",
  });
}

describe("archiveLibraries", () => {
  it("archives each library, holding the ones templates import for confirmation", async () => {
    const archive = vi.fn(async (id: string, confirmInUse: boolean) => {
      if (id === "fl_used" && !confirmInUse) throw needsConfirmation();
      return { id };
    });

    const outcome = await archiveLibraries(["fl_free", "fl_used"], false, archive);

    expect(archive.mock.calls).toEqual([
      ["fl_free", false],
      ["fl_used", false],
    ]);
    expect(outcome).toEqual({
      archived: 1,
      failed: 0,
      held: [{ item: "fl_used", message: inUse }],
      forbidden: [],
    });
  });

  it("sends the confirmation once the person confirms, and counts a refusal then as failed", async () => {
    const archive = vi.fn(async () => {
      throw needsConfirmation();
    });

    const outcome = await archiveLibraries(["fl_used"], true, archive);

    expect(archive).toHaveBeenCalledWith("fl_used", true);
    expect(outcome).toEqual({ archived: 0, failed: 1, held: [], forbidden: [] });
  });

  it("reports a permission refusal in the server's words and keeps going", async () => {
    const archive = vi.fn(async (id: string) => {
      if (id === "fl_a") throw forbidden();
      if (id === "fl_b") throw new Error("network");
      return { id };
    });

    const outcome = await archiveLibraries(["fl_a", "fl_b", "fl_c"], false, archive);

    expect(outcome).toEqual({
      archived: 1,
      failed: 2,
      held: [],
      forbidden: ["Archiving a library templates import needs approve permission"],
    });
  });
});
